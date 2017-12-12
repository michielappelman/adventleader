package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

var githash = "No Git hash provided"
var buildstamp = "No build timestamp provided"

const timeLayout = "2006-01-02T15:04:05-0700"

// Defines the struct for use with the JSON config file.
type Configuration struct {
	Debug    bool
	URL      string
	Cookie   string
	BotToken string
	RoomID   string
}

// Create custom time struct for ISO8601 type in JSON
type JSONTime struct {
	time.Time
}

func (t *JSONTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		t.Time = time.Time{}
		return
	}
	t.Time, err = time.Parse(timeLayout, s)
	return
}

// Define the Leaderboard JSON structure
type Leaderboard struct {
	OwnerID string            `json:"owner_id"`
	Event   string            `json:"event"`
	Members map[string]Member `json:"members"`
}

type Member struct {
	ID          string                      `json:"id"`
	Name        string                      `json:"name"`
	Stars       int                         `json:"stars"`
	LocalScore  int                         `json:"local_score"`
	GlobalScore int                         `json:"global_score"`
	LastStarTS  JSONTime                    `json:"last_star_ts"`
	Days        map[string]map[string]Level `json:"completion_day_level"`
}

type Level struct {
	Timestamp JSONTime `json:"get_star_ts"`
}

type SortedMembers []Member

func (m SortedMembers) Len() int      { return len(m) }
func (m SortedMembers) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m SortedMembers) Less(i, j int) bool {
	if m[i].LocalScore < m[j].LocalScore {
		return true
	}
	if m[i].LocalScore > m[j].LocalScore {
		return false
	}
	return m[i].Stars < m[j].Stars
}

// Uses the URL and cookie to retrieve a Leaderboad struct
func GetLeaderboard(url, cookie string) Leaderboard {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	c := http.Cookie{Name: "session", Value: cookie}
	req.AddCookie(&c)

	var client = &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	// Unmarshal the JSON data to the Leaderboard struct
	var b Leaderboard
	if err := json.Unmarshal(body, &b); err != nil {
		log.Fatal(err)
	}
	return b
}

func JSONtoNormalTime(jt JSONTime) time.Time {
	t, err := time.Parse(time.RFC3339, jt.Format(time.RFC3339))
	if err != nil {
		log.Println(err)
	}
	return t
}

func PostToSpark(botToken, roomID, message string) int {
	const message_api = "https://api.ciscospark.com/v1/messages"

	jsonData := struct {
		Room string `json:"roomId"`
		Msg  string `json:"markdown"`
	}{
		roomID,
		message,
	}
	data, err := json.Marshal(jsonData)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", message_api, bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+botToken)
	req.Header.Set("Content-type", "application/json; charset=utf-8")

	var client = &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	return resp.StatusCode
}

func MainLoop(config Configuration, lastUpdate time.Time) time.Time {
	leaderboard := GetLeaderboard(config.URL, config.Cookie)

	// Create a slice of ints and a map with values/keys
	// based on the local score of the Members
	var lastStar time.Time
	var members []Member

	for _, member := range leaderboard.Members {
		members = append(members, member)
		memberStarTime := JSONtoNormalTime(member.LastStarTS)
		if memberStarTime.After(lastStar) {
			lastStar = memberStarTime
		}
	}

	sort.Sort(sort.Reverse(SortedMembers(members)))

	lb := strings.TrimSuffix(config.URL, ".json")
	message := fmt.Sprintf("### [Leaderboard 🎄](%s)\n\n---\n", lb)
	// Create message with list of Members sorted by Local Score
	for _, m := range members {
		var name string
		if m.Name == "" {
			name = m.ID
		} else {
			name = m.Name
		}
		if m.Stars > 0 {
			message += fmt.Sprintf(" 1. 📈 `%03d` ⭐ `%02d` – **%s** ",
				m.LocalScore, m.Stars, name)
		}
		if m.GlobalScore > 0 {
			message += fmt.Sprintf(" (🌍 _%d_!)", m.GlobalScore)
		}
		message += "\n"
	}
	if config.Debug {
		fmt.Print(message)
	}

	// Send a message only if there was a Star gotten after the last Update
	if lastStar.After(lastUpdate) || config.Debug {
		_ = PostToSpark(config.BotToken, config.RoomID, message)
	}
	return time.Now()
}

// Main method
func main() {
	fmt.Println("Git Commit Hash:", githash)
	fmt.Println("UTC Build Time :", buildstamp)
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}

	var config Configuration
	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Fatal(err)
	}

	lastUpdate := time.Now()
	for {
		lastUpdate = MainLoop(config, lastUpdate)
		time.Sleep(300 * time.Second)
	}
}
