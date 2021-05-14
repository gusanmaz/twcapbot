## twcapbot - Tweet Caption Bot and CLI

twcapbot provides two distinct programs.

* **tweet-captioner-cli** CLI tool allows anyone with Twitter API keys to retrieve media of recent (latest 3200 tweets and favorites) tweets of a specific user. Aside from original pictures from tweets, you could retrieve captioned pictures of these pictures. Caption text includes tweet text and some other useful stuff for regarding tweet.
* **tweet-captioner-bot** When this bot is working users could retrieve captioned images as reply for a particular tweet by simply mentioning this bot's host account.

## Installation

1. `go get github.com/gusanmaz/twcapbot`

2. `cd ${GOPATH}/src/github.com/gusanmaz/twcapbot`

3. `go install cmd/tweet-caption-bot/*`

4. `go install cmd/tweet-caption-cli/*`
## Usage 

### tweet-captioner-cli

Command below obtains recent (latest 3200) favorites of @github account and saves media, and captioned media of these tweets into . (current directory)

`tweet-captioner-cli -creds creds.json -o . -s github -type fav`

* To obtain tweets of a user instead of favorites change `type fav` into `type tweet`
* Twitter API credentials are stored in a file and this file's location should be provided as cred flag's value
* We will present an empty credentials file below. Once you obtain Twitter API credentials you could modify this file according to your API keys.
* All output of the command is saved into directory determined by -o flag value.

### tweet-captioner-bot

Usage of bot CLI similar but simpler.

`tweet-captioner-bot -creds creds.json -o .`

## Twitter API Credentials File

Change values of the credentials JSON files according to your API keys.

```json
{
  "APIKey": "",
  "APISecret": "",
  "bearerToken": "",
  "accessToken": "",
  "accessSecret": ""
}
```

## Author

Güvenç Usanmaz

## License

MIT License