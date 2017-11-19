package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const debug bool = true
const timeLayout = "2006-01-02T15:04:05-0700"

type Configuration struct {
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
	var keys []int
	var lastStar time.Time
	ids := make(map[int]Member)

	for _, member := range leaderboard.Members {
		keys = append(keys, member.LocalScore)
		ids[member.LocalScore] = member
		memberStarTime := JSONtoNormalTime(member.LastStarTS)
		if memberStarTime.After(lastStar) {
			lastStar = memberStarTime
		}
	}

	// Sort the slice of Ints in reverse order
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))

	message := "### Leaderboard 🎄\n\n"
	// Create message with list of Members sorted by Local Score
	for _, k := range keys {
		var name string
		if ids[k].Name == "" {
			name = ids[k].ID
		} else {
			name = ids[k].Name
		}
		message += " 1. **" + name + "**"
		message += " 📈 _" + strconv.Itoa(ids[k].LocalScore) + "_"
		message += " ⭐ _" + strconv.Itoa(ids[k].Stars) + "_"
		message += "\n"
	}
	if debug {
		fmt.Print(message)
	}

	// Send a message only if there was a Star gotten after the last Update
	if lastStar.After(lastUpdate) || debug {
		_ = PostToSpark(config.BotToken, config.RoomID, message)
		lastUpdate = time.Now()
	}
	return lastUpdate
}

// Main method
func main() {
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
