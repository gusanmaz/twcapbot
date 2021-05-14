package twcapbot

import (
	"embed"
	"fmt"
	"github.com/gusanmaz/capdec"
	"github.com/gusanmaz/twigger"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//go:embed hair.png
var embedFS embed.FS

type TweetCaptionBot struct {
	JSCodes       []string
	OutDirPath    string
	TwiggerConn   *twigger.Connection
	HairPhotoPath string
	InfoLog       *log.Logger
	ErrLog        *log.Logger
}

const botLogPrefix = "Tweet Caption Bot: "
const DownloadRetries = 5

func init() {
	//capdec.ChangeMaxBrowserDimensions(5500, 3200)
}

func New(creds twigger.Credentials, logFile *os.File, codes []string, outDirPath string) *TweetCaptionBot {
	bot := TweetCaptionBot{}
	bot.JSCodes = codes
	bot.OutDirPath = outDirPath

	finfo, err := os.Stat(outDirPath)

	if err != nil || !finfo.IsDir() {
		log.Panicf("%v is not a valid directory", outDirPath)
	}

	tConn, err := twigger.NewConnection(creds, logFile, os.Stdout, os.Stderr)
	if err != nil {
		log.Panicf("Cannot create a new connection for twigger. Error: %v", err)
	}
	bot.TwiggerConn = tConn

	f, err := embedFS.Open("hair.png")
	if err != nil {
		log.Panicf("Cannot open hair.png. Error: %v", err)
	}

	tempF, err := ioutil.TempFile(os.TempDir(), "hair.*.png")
	if err != nil {
		log.Panicf("Cannot create temporary file for hair.png. Error: %v", err)
	}

	_, err = io.Copy(tempF, f)
	if err != nil {
		log.Panicf("Cannot copy hair.png into temporary directory. Error: %v", err)
	}

	infoW := tConn.ErrLog.Writer()
	bot.InfoLog = log.New(infoW, botLogPrefix, log.LstdFlags)

	errW := tConn.ErrLog.Writer()
	bot.ErrLog = log.New(errW, botLogPrefix, log.LstdFlags)

	bot.HairPhotoPath = filepath.Join(tempF.Name())
	return &bot
}

func (b *TweetCaptionBot) CaptionTweet(id int64, rootPath string) error {
	tw, err := b.TwiggerConn.GetSingleTweetFromID(id)
	if err != nil {
		return err
	}

	var quotedTweet *twigger.Tweet = nil
	if tw.QuotedStatusID != 0 {
		tw, err := b.TwiggerConn.GetSingleTweetFromID(tw.QuotedStatusID)
		if err != nil {
			return err
		}
		quotedTweet = &tw
	}

	fNameInfo := GenerateFileNamesForTweet(tw, quotedTweet)
	userDirPath := filepath.Join(rootPath, fNameInfo[0].ShortDirName)

	_, err = os.Stat(rootPath)
	if err != nil {
		err = os.Mkdir(rootPath, 0750)
		if err != nil {
			b.ErrLog.Printf("Cannot create directory: %v! Error message: %v", userDirPath, err)
			panic("Exiting program...")
		}
	}

	_, err = os.Stat(userDirPath)
	if err != nil {
		err = os.Mkdir(userDirPath, 0750)
		if err != nil {
			b.ErrLog.Printf("Cannot create directory: %v! Error message: %v", userDirPath, err)
			panic("Exiting program...")
		}
	}

	for _, v := range fNameInfo {
		srcPath := filepath.Join(userDirPath, v.LongFileName)
		destFilePath := filepath.Join(userDirPath, v.LongCaptionFileName)
		b.InfoLog.Printf("Captioning of tweet with ID of %v has started", tw.Id)
		if v.MediaTweet {
			err := DownloadTo(v.MediaURL, srcPath)
			if err != nil {
				b.InfoLog.Printf("Download of media files of tweet with ID %v has failed!", tw.Id)
				for i := 1; i <= DownloadRetries; i++ {
					b.InfoLog.Printf("Attempt %v/%v to download media files of tweet (ID: %v)", i+1, DownloadRetries, tw.Id)
					err = DownloadTo(v.MediaURL, srcPath)
					if err == nil {
						b.InfoLog.Printf("Media files for tweet (ID: %v) has succesfully downloaded", tw.Id)
						break
					}
				}
				if err != nil {
					b.ErrLog.Printf("%v attempts to download media files for tweet with ID of %v has failed!", DownloadRetries)
					return err
				}
			}
			err = capdec.Caption(srcPath, GetCaptionsForTweet(tw, quotedTweet), destFilePath, b.JSCodes)
			if err != nil {
				b.ErrLog.Printf("Captioning of tweet with ID of %v is unsuccessful!", tw.Id)
				b.ErrLog.Printf("Error message: %v", err)
				panic("Exiting program...")
			}
		} else {
			err := capdec.Caption(b.HairPhotoPath, GetCaptionsForTweet(tw, quotedTweet), destFilePath, b.JSCodes)
			if err != nil {
				b.ErrLog.Printf("Captioning of tweet with ID of %v is unsuccessful!", tw.Id)
				b.ErrLog.Printf("Error message: %v", err)
				panic("Exiting program...")
			}
		}
		b.InfoLog.Printf("Captioning of tweet with ID of %v has completed successfully", tw.Id)
	}

	htmlFilePath := filepath.Join(userDirPath, fNameInfo[0].LongHTMLFileName)
	htmFile, err := os.OpenFile(htmlFilePath, os.O_RDWR|os.O_CREATE, 0644)
	defer htmFile.Close()
	if err != nil {
		b.ErrLog.Printf("Couldn't create %v", tw.Id)
	}

	templateStr := `<script> location.href = "{{}}" </script>`
	text := strings.Replace(templateStr, "{{}}", GetTweetURL(tw), 1)

	_, err = htmFile.WriteString(text)
	if err != nil {
		b.ErrLog.Printf("Couldn't write tweet (ID: %v) url into %v", tw.Id, htmlFilePath)
	}

	return nil
}

func GetTweetURL(tw twigger.Tweet) string {
	sn := tw.User.ScreenName
	url := fmt.Sprintf("https://www.twitter.com/%v/status/%v", sn, tw.Id)
	return url
}
