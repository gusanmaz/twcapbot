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

const(
	credsDef = "creds.json"
	credsUsage    = "File path for Twitter API credentials. File should be in JSON or YAML format"

	screenNameDef = "bbcnews"
	screenNameUsageUsage = "User name of the Twitter user"

	tweetTypeDef  = "tweet"
	tweetTypeUsage = "Type of the tweets that will be captioned. Valid values: tweet, favorite, fav"

	logFileDef   = "bot.log"
	logFileUsage = "name of the bot file"

	outPathDefUsage = "Output directory for saving original tweet media and captioned tweet photos"

	shortcut = " (shortcut)"
)

var(
	credsFlag string
	screenNameFlag string
	tweetTypeFlag string
	outPathFlag string
	logFileFlag string
)

func main(){
	homeDir, err := os.UserHomeDir()
	if err != nil{
		log.Panicf("Home directory is not defined! Error message: %v", err)
	}
	outPathDef := filepath.Join(homeDir, "tweet_caption_bot")

	flag.StringVar(&credsFlag, "creds", credsDef, credsUsage)
	flag.StringVar(&credsFlag, "c", credsDef, credsUsage + shortcut)

	flag.StringVar(&screenNameFlag, "screenName", screenNameDef, screenNameUsageUsage)
	flag.StringVar(&screenNameFlag, "s", screenNameDef, screenNameUsageUsage + shortcut)

	flag.StringVar(&tweetTypeFlag, "type", tweetTypeDef, tweetTypeUsage)
	flag.StringVar(&tweetTypeFlag, "t", tweetTypeDef, tweetTypeUsage + shortcut)

	flag.StringVar(&outPathFlag, "out", outPathDef, outPathDefUsage)
	flag.StringVar(&outPathFlag, "o", outPathDef, outPathDefUsage + shortcut)

	flag.StringVar(&logFileFlag, "log", logFileDef, logFileUsage)
	flag.StringVar(&logFileFlag, "l", logFileDef, logFileUsage + shortcut)

	flag.Parse()

	logFilePath := filepath.Join(outPathFlag, logFileFlag)
	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil{
		log.Panicf("Log file %v couldn't be created. Error message: %v", logFilePath, err)
	}
	defer f.Close()

	creds, err := twigger.LoadCredentials(credsFlag)
	if err != nil{
		log.Panicf("Credentials file %v couldn't be loaded. Error message: %v", credsFlag, err)
	}

	finfo, err := os.Stat(outPathFlag)
	if err != nil || finfo.IsDir() == false{
		log.Panicf("Given output directory: %v is not valid!", err)
	}

	bot := twcapbot.New(creds, f , []string{""}, outPathFlag)

	twiggerFunc := bot.TwiggerConn.GetAllRecentTweetsFromScreenName
	tweetType   := "tweets"
	if strings.Contains(strings.ToLower(tweetTypeFlag), "fav"){
		twiggerFunc = bot.TwiggerConn.GetAllRecentFavoritesFromScreenName
		tweetType = "favorites"
	}

	timeName := time.Now().Format("01_02_2006_15_04")
	jsonFileName := fmt.Sprintf("%v_%v_%v.json", screenNameFlag, timeName, tweetType)
	jsonFilePath := filepath.Join(bot.OutDirPath, jsonFileName)

	tweets, err := twiggerFunc(screenNameFlag)
	if err != nil{
		log.Printf("Retrieval of recent %v of user %v has failed!", tweetType, screenNameFlag)
		log.Panicf("Error message: %v", err)
	}

	err = tweets.Save(jsonFilePath)
	bot.ErrLog.Printf("Saving of recent %v of user %v to %v has failed!", tweetType, screenNameFlag, jsonFilePath)

	twUserDirName := fmt.Sprintf("%v_%v_%v_%v",
		bot.TwiggerConn.User.ScreenName, bot.TwiggerConn.User.Id, tweetType, timeName)
	captionRootDir := filepath.Join(bot.OutDirPath, twUserDirName)

	for i, tw := range tweets{
		bot.InfoLog.Printf("Tweet captioning task %v/%v has started", i + 1, len(tweets))
		err := bot.CaptionTweet(tw.Id, captionRootDir)

		if err == nil{
			bot.InfoLog.Printf("Tweet captioning task %v/%v has completed successfully", i + 1, len(tweets))
		}else{
			bot.InfoLog.Printf("Tweet captioning task %v/%v has failed", i + 1, len(tweets))
		}
	}
}
