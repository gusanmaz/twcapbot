package main

import (
	"flag"
	"fmt"
	"github.com/gusanmaz/twcapbot"
	"github.com/gusanmaz/twigger"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FailEvent struct {
	Retry int
	Time  int64
	Error error
}

type Mention struct {
	IDStr    string
	ID       int64
	Time     int64
	Tweet    twigger.Tweet
	Failures []FailEvent
}

type SafeTasks struct {
	Tasks map[string]Mention
	mu    sync.Mutex
}

const (
	credsDef   = "creds.json"
	credsUsage = "File path for Twitter API credentials. File should be in JSON or YAML format"

	logFileDef   = "bot.log"
	logFileUsage = "name of the bot file"

	outPathDefUsage = "Output directory for saving original tweet media and captioned tweet photos"

	shortcut          = " (shortcut)"
	selfReferenceText = "foo(goo())"

	TestBot              = true // Set true if bot account and test account is the same one
	ResponseText         = "Your captioned tweet is ready!"
	MentionQueryPause    = 12 * time.Second // in seconds
	MaxRetrievalAttempts = 10
	ReplyWindow          = 20 * time.Minute // If bot cannot reply in ReplyWindow discard that tweet
)

var (
	credsFlag          string
	outPathFlag        string
	logFileFlag        string
	FailedTasksPath    string
	CompletedTasksPath string
	Tasks              SafeTasks
	sinceID            int64
	finished           chan bool
)

func AppendToFile(path, text string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	_, err = f.WriteString(text + "\n")
	f.Close()
}

func AppendToFailedTasksFile(text string) {
	AppendToFile(FailedTasksPath, text)
}

func AppendToCompletedTasksFile(text string) {
	AppendToFile(CompletedTasksPath, text)
}

func ReplyToMention(bot *twcapbot.TweetCaptionBot, tw twigger.Tweet) (int64, error) {
	conn := bot.TwiggerConn
	mentionText := tw.FullText
	bot.InfoLog.Printf("Preparation of caption tweet for tweet (User: %v ID: %v) has started.", tw.User.ScreenName, tw.Id)
	if TestBot && (strings.Contains(mentionText, ResponseText) || strings.Contains(mentionText, selfReferenceText)) {
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

func GetNewMentions(bot twcapbot.TweetCaptionBot) {
	fmt.Println("NEW MENTIONS")
	i := 0
	var mentions twigger.Tweets
	var err error

	maxID := sinceID

	for ; i < MaxRetrievalAttempts; i++ {
		mentions, err = bot.TwiggerConn.GetRecentNMentionsSince(200, sinceID)
		if err == nil {
			break
		}
	}
	if err == nil {
		bot.InfoLog.Printf("Retrieval of %v mention tweets has succeeded at %v. attempt.\n", len(mentions), i+1)
	} else {
		bot.ErrLog.Println("Retrieval of mention tweets has failed!")
	}

	Tasks.mu.Lock()
	for _, mention := range mentions {
		if mention.Id > maxID {
			maxID = mention.Id
		}
		t, err := time.Parse(time.RubyDate, mention.CreatedAt)
		mentionTime := t.Unix()
		if err != nil {
			mentionTime = time.Now().Unix()
		}
		Tasks.Tasks[mention.IdStr] = Mention{
			IDStr:    mention.IdStr,
			ID:       mention.Id,
			Time:     mentionTime,
			Failures: []FailEvent{},
			Tweet:    mention,
		}
	}
	sinceID = maxID
	Tasks.mu.Unlock()
	time.Sleep(MentionQueryPause)
	bot.TwiggerConn.Reconnect()
}

func ReplyToNextMention(bot twcapbot.TweetCaptionBot) {
	curKey := ""
	curMention := Mention{}
	tweet := twigger.Tweet{}
	Tasks.mu.Lock()
	maxRetry := 1000
	for i, v := range Tasks.Tasks {
		if len(v.Failures) < maxRetry {
			curKey = i
			curMention = v
			tweet = v.Tweet
			maxRetry = len(v.Failures)
		}
		if maxRetry == 0 {
			break
		}
	}
	Tasks.mu.Unlock()

	if curKey == "" {
		return
	}

	now := time.Now()
	nowString := now.String()
	requestTime := time.Unix(curMention.Time, 0)
	waitDuration := now.Sub(requestTime)

	tweetID := tweet.IdStr
	handle := tweet.User.ScreenName

	if waitDuration > ReplyWindow {
		text := strings.Join([]string{nowString, handle, tweetID, "Reply discarded because of timeout"}, ",")
		AppendToFailedTasksFile(text)
		bot.ErrLog.Printf(text)
		failures := curMention.Failures
		for _, failure := range failures {
			text := strings.Join([]string{fmt.Sprintf("%v", time.Unix(failure.Time, 0).String()),
				failure.Error.Error()}, ",")
			text = fmt.Sprintf("Failure #%v: ", failure.Retry) + text
			AppendToFailedTasksFile(text)
		}
		return
	}

	source := tweet.Source
	if strings.Contains(source, bot.TwiggerConn.User.Name) {
		text := strings.Join([]string{nowString, handle, tweetID, "Reply discarded because this tweet is generated by the same bot."}, ",")
		AppendToFailedTasksFile(text)
		bot.InfoLog.Printf(text)
		return
	}

	replyID, err := ReplyToMention(&bot, curMention.Tweet)

	retry := 1
	if curMention.Failures != nil {
		retry = len(curMention.Failures) + 1
	}
	if err != nil {
		failure := FailEvent{
			Retry: retry,
			Time:  time.Now().Unix(),
			Error: err,
		}
		curMention.Failures = append(curMention.Failures, failure)
	} else {
		Tasks.mu.Lock()
		delete(Tasks.Tasks, curKey)
		bot.InfoLog.Printf("Reply for tweet#%v has been published as tweet#%v", curMention.ID, replyID)
		now := time.Now().String()
		text := strings.Join([]string{now, curMention.Tweet.User.ScreenName, curMention.Tweet.User.IdStr,
			curMention.IDStr, fmt.Sprintf("%v", replyID)}, ",")
		AppendToCompletedTasksFile(text)
		Tasks.mu.Unlock()
	}
}

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

	FailedTasksPath = path.Join(outPathFlag, "fail.tasks")
	CompletedTasksPath = path.Join(outPathFlag, "success.tasks")

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
	Tasks.Tasks = make(map[string]Mention)

	// If your Twitter account zero mention tweets bot would fail!
	mentions, err := bot.TwiggerConn.GetRecentNMentions(1)
	if err != nil || len(mentions) != 1 {
		log.Panicf("Cannot retrieve the last mention for %v", bot.TwiggerConn.User.ScreenName)
	}
	sinceID = mentions[0].Id

	infGetNewMentions := func(id int, wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			GetNewMentions(*bot)
		}
	}

	infReplyToNextMention := func(id int, wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			ReplyToNextMention(*bot)
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go infGetNewMentions(1, &wg)
	wg.Add(1)
	go infReplyToNextMention(2, &wg)

	wg.Wait()
}
