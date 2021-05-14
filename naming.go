package twcapbot

import (
	"fmt"
	"github.com/gusanmaz/twigger"
	"path/filepath"
)

const mediaIDWidth = 16

type TweetFileNameInfo struct {
	LongFileName  string
	ShortFileName string
	ShortDirName  string

	LongCaptionFileName  string
	ShortCaptionFileName string
	ShortCaptionDirName  string

	LongHTMLFileName  string
	ShortHTMLFileName string

	MediaTweet bool
	MediaURL   string
	MediaName  string
}

func GenerateFileNamesForTweet(tw twigger.Tweet, quotedTweet *twigger.Tweet) []TweetFileNameInfo {
	if quotedTweet != nil {
		parentFileNames := GenerateFileNamesForTweet(tw, nil)
		childFileNames := GenerateFileNamesForTweet(*quotedTweet, nil)
		parent0 := parentFileNames[0]
		child0 := childFileNames[0]

		if parent0.MediaTweet == false && child0.MediaTweet == false {
			return parentFileNames
		} else if parent0.MediaTweet == true && child0.MediaTweet == false {
			return parentFileNames
		} else if parent0.MediaTweet == false && child0.MediaTweet == true {
			ret := make([]TweetFileNameInfo, len(childFileNames))
			for i, v := range childFileNames {
				ret[i] = TweetFileNameInfo{
					LongFileName:         v.LongFileName,
					ShortFileName:        v.ShortFileName,
					LongHTMLFileName:     v.LongHTMLFileName,
					ShortHTMLFileName:    v.ShortHTMLFileName,
					ShortDirName:         parent0.ShortDirName,
					LongCaptionFileName:  v.LongFileName,
					ShortCaptionFileName: v.ShortCaptionFileName,
					ShortCaptionDirName:  parent0.ShortCaptionDirName,
					MediaTweet:           true,
					MediaURL:             v.MediaURL,
					MediaName:            v.MediaName,
				}
			}
			return ret
		} else {
			return parentFileNames
		}
	}

	urls := tw.GetMediaURLs()

	mediaNames := make([]string, len(urls))
	for i, url := range urls {
		mediaNames[i] = filepath.Base(url)[:mediaIDWidth]
	}

	userName := fmt.Sprintf("%v_%v", tw.User.Id, tw.User.ScreenName)

	if len(mediaNames) == 0 {
		ret := make([]TweetFileNameInfo, 1)
		info := TweetFileNameInfo{
			LongFileName:         fmt.Sprintf("%v_%v.png", userName, tw.Id),
			ShortFileName:        fmt.Sprintf("%v.png", tw.Id),
			LongHTMLFileName:     fmt.Sprintf("%v_%v.html", userName, tw.Id),
			ShortHTMLFileName:    fmt.Sprintf("%v.html", tw.Id),
			ShortDirName:         userName,
			LongCaptionFileName:  fmt.Sprintf("%v_%v_caption.png", userName, tw.Id),
			ShortCaptionFileName: fmt.Sprintf("%v_caption.png", tw.Id),
			ShortCaptionDirName:  userName + "_caption",
			MediaTweet:           false,
			MediaURL:             "",
			MediaName:            "",
		}
		info.LongFileName = ""
		info.ShortFileName = ""
		ret[0] = info
		return ret
	}

	ret := make([]TweetFileNameInfo, len(mediaNames))

	for i, mediaName := range mediaNames {
		info := TweetFileNameInfo{
			LongFileName:         fmt.Sprintf("%v_%v_%v.png", userName, tw.Id, mediaName),
			ShortFileName:        fmt.Sprintf("%v_%v", tw.Id, mediaName),
			LongHTMLFileName:     fmt.Sprintf("%v_%v.html", userName, tw.Id),
			ShortHTMLFileName:    fmt.Sprintf("%v.html", tw.Id),
			ShortDirName:         userName,
			LongCaptionFileName:  fmt.Sprintf("%v_%v_%v_caption.png", userName, tw.Id, mediaName),
			ShortCaptionFileName: fmt.Sprintf("%v_%v_caption.png", tw.Id, mediaName),
			ShortCaptionDirName:  userName + "_caption",
			MediaTweet:           true,
			MediaURL:             urls[i],
			MediaName:            mediaName,
		}
		ret[i] = info
	}
	return ret
}
