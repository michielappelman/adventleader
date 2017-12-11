# AdventLeader Bot

This bot retrieves the JSON of your private leaderboard of the 
[Advent of Code](http://adventofcode.com/) and posts it to the Spark room of your choosing.

## Cisco Spark Requirements

To create a Spark bot, visit [My Apps](https://developer.ciscospark.com/apps.html) in the Cisco
Spark portal.

Add that Bot to the room of your choosing and using the Bot Token, retrieve the Room ID. For 
example by using [httpie](https://httpie.org/):

```
http -j GET "https://api.ciscospark.com/v1/rooms" "Authorization: Bearer <bot token>"
```

Finally you'll need a session cookie from the Advent of Code website. Simply open your browser's
Web Developer tools and look for the _Cookie_ tab in the _Network_ panel.

When you have all these inputs, fill the `config.json` file, like so:

```json
{
    "Debug": true,
    "URL": "<AoC JSON URL>",
    "Cookie": "<AoC Session Cookie>",
    "BotToken": "<Spark Bot Token>",
    "RoomID": "<Spark Room ID>"
}
```

If everything went well, your bot should start posting to your channel everytime someone gets a
new star. Note that the interval between checks is set to 5 minutes, to not overload the AoC
servers.

Happy coding! 🎅
