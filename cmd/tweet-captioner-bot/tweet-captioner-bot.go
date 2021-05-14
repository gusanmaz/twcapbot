package main

import (
	"flag"
	"fmt"
	"github.com/gusanmaz/twcapbot"
	"github.com/gusanmaz/twigger"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	credsDef   = "creds.json"
	credsUsage = "File path for Twitter API credentials. File should be in JSON or YAML format"

	logFileDef   = "bot.log"
	logFileUsage = "name of the bot file"

	outPathDefUsage = "Output directory for saving original tweet media and captioned tweet photos"

	shortcut = " (shortcut)"
)

const TestBot = true // Set true if bot account and test account is the same one
const ResponseText = "here is the captioned tweet you've requested!"
const BatchInterval = 12 // in seconds

var (
	credsFlag   string
	outPathFlag string
	logFileFlag string
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Panicf("Home directory is not defined! Error message: %v", err)
	}
	outPathDef := filepath.Join(homeDir, "tweet_caption_bot")

	flag.StringVar(&credsFlag, "creds", credsDef, credsUsage)
	flag.StringVar(&credsFlag, "c", credsDef, credsUsage+shortcut)

	flag.StringVar(&outPathFlag, "out", outPathDef, outPathDefUsage)
	flag.StringVar(&outPathFlag, "o", outPathDef, outPathDefUsage+shortcut)

	flag.StringVar(&logFileFlag, "log", logFileDef, logFileUsage)
	flag.StringVar(&logFileFlag, "l", logFileDef, logFileUsage+shortcut)

	flag.Parse()

	logFilePath := filepath.Join(outPathFlag, logFileFlag)
	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Panicf("Log file %v couldn't be created. Error message: %v", logFilePath, err)
	}
	defer f.Close()

	creds, err := twigger.LoadCredentials(credsFlag)
	if err != nil {
		log.Panicf("Credentials file %v couldn't be loaded. Error message: %v", credsFlag, err)
	}

	finfo, err := os.Stat(outPathFlag)
	if err != nil || finfo.IsDir() == false {
		log.Panicf("Given output directory: %v is not valid!", err)
	}

	bot := twcapbot.New(creds, f, []string{""}, outPathFlag)
	twcapbot.SetBotScreenName(bot.TwiggerConn.User.ScreenName)

	// If your Twitter account zero mention tweets bot would fail!
	mentions, err := bot.TwiggerConn.GetRecentNMentions(1)
	if err != nil || len(mentions) != 1 {
		log.Panicf("Cannot retrieve the last mention for %v", bot.TwiggerConn.User.ScreenName)
	}
	sinceID := mentions[0].Id + 1

	counter := 1
	for {
		bot.InfoLog.Printf("Batch work #%v has started", counter)
		sinceID, err = BatchWork(bot, sinceID)
		if err != nil {
			bot.ErrLog.Printf("Batch work #%v has failed!", counter)
			bot.ErrLog.Printf("Batch work #%v error message: %v", counter, err)
		} else {
			bot.InfoLog.Printf("Batch work #%v has completed successfully", counter)
		}
		time.Sleep(time.Second * BatchInterval)
		counter++
	}
}

func BatchWork(bot *twcapbot.TweetCaptionBot, sinceID int64) (int64, error) {
	mentions, err := bot.TwiggerConn.GetAllRecentMentionsSince(sinceID)
	maxID := sinceID
	if err != nil {
		bot.ErrLog.Printf("An error occurred during retrieval of recent mentions after ID: %v", sinceID)
		bot.ErrLog.Printf("Recent mentions retrieval error: %v", err)
	}
	for _, mentionsTweet := range mentions {
		id, err := ReplyToMention(bot, mentionsTweet)
		if err != nil {
			return maxID, err
		}
		if id > maxID {
			maxID = id
		}
	}
	return maxID, nil
}

func ReplyToMention(bot *twcapbot.TweetCaptionBot, tw twigger.Tweet) (int64, error) {
	conn := bot.TwiggerConn
	mentionText := tw.FullText
	bot.InfoLog.Printf("Preparation of caption tweet for tweet (User: %v ID: %v) has started.", tw.User.ScreenName, tw.Id)
	if TestBot && (strings.Contains(mentionText, ResponseText) || strings.Contains(mentionText, "abakus!")) {
		bot.InfoLog.Println("Mention tweet is skipped for captioning because of self-reference")
		return -1, nil
	}

	realTweetID := tw.InReplyToStatusID
	err := bot.CaptionTweet(realTweetID, outPathFlag)
	if err != nil {
		bot.ErrLog.Println("Error: %v", err)
		return -1, err
	}
	realTweet, err := conn.GetSingleTweetFromID(realTweetID)
	if err != nil {
		bot.ErrLog.Println("Error: %v", err)
		return -1, err
	}
	var quotedTweet *twigger.Tweet = nil
	if realTweet.QuotedStatusID != 0 {
		*quotedTweet, err = conn.GetSingleTweetFromID(realTweet.QuotedStatusID)
		if err != nil {
			bot.ErrLog.Println("Error: %v", err)
			return -1, err
		}
	}

	fileNames := twcapbot.GenerateFileNamesForTweet(realTweet, quotedTweet)
	pathNames := make([]string, len(fileNames))

	for i, v := range fileNames {
		pathNames[i] = filepath.Join(outPathFlag, v.ShortDirName, v.LongCaptionFileName)
	}
	personalizedResponseText := fmt.Sprintf("@%v %v", tw.User.ScreenName, ResponseText)
	respID, err := conn.PublishCollageTweetAsReply(pathNames, personalizedResponseText, tw.Id)
	if err != nil {
		bot.ErrLog.Println("Error: %v", err)
		return -1, err
	}
	bot.InfoLog.Printf("Caption tweet for tweet (User: %v ID: %v) has been just published.", tw.User.ScreenName, tw.Id)
	bot.InfoLog.Printf("ID of newly published caption tweet is %v", respID)
	return respID, nil
}
