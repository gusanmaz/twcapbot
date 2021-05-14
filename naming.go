package twcapbot

import (
	"fmt"
	"github.com/gusanmaz/twigger"
	"path"
	"path/filepath"
)

const mediaIDWidth = 16

type TweetFileNameInfo struct{
	LongFileName string
	ShortFileName string
	ShortDirName string

	LongCaptionFileName string
	ShortCaptionFileName string
	ShortCaptionDirName string

	MediaTweet   bool
	MediaURL     string
	MediaName    string
}

func GenerateFileNamesForTweet(tw twigger.Tweet) []TweetFileNameInfo{
	urls := tw.GetMediaURLs()

	mediaNames := make([]string, len(urls))
	for i, url := range urls{
		mediaNames[i] = filepath.Base(url)[:mediaIDWidth]
	}

	userName := fmt.Sprintf("%v_%v", tw.User.Id, tw.User.ScreenName)

	if len(mediaNames) == 0{
		ret := make([]TweetFileNameInfo, 1)
		info := TweetFileNameInfo{
			LongFileName:         fmt.Sprintf("%v_%v.png", userName, tw.Id),
			ShortFileName:        fmt.Sprintf("%v.png", tw.Id),
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

	for i, mediaName := range mediaNames{
		info := TweetFileNameInfo{
			LongFileName:         fmt.Sprintf("%v_%v_%v.png", userName, tw.Id, mediaName),
			ShortFileName:        fmt.Sprintf("%v_%v", tw.Id, mediaName),
			ShortDirName:         userName,
			LongCaptionFileName:  fmt.Sprintf("%v_%v_%v_caption.png", userName, tw.Id, mediaName),
			ShortCaptionFileName: fmt.Sprintf("%v_%v_caption.png", tw.Id,mediaName),
			ShortCaptionDirName:  userName + "_caption",
			MediaTweet:           true,
			MediaURL:             urls[i],
			MediaName:            mediaName,
		}
		ret[i] = info
	}
	return ret
}

func GetLongFileNameMappingForMediaTweet(tw twigger.Tweet, captionMedia bool)map[string]string{
	mediaurls := tw.GetMediaURLs()
	baseFilenames := make([]string, len(mediaurls))
	longFilenames  := make([]string, len(mediaurls))

	for i, url := range mediaurls{
		baseFilenames[i] = path.Base(url)
	}

	captionText := ""
	if captionMedia{
		captionText = "_cap_"
	}
	fnMap := map[string]string{}
	for i, filename := range baseFilenames{
		longFilenames[i] = fmt.Sprintf("%v_%v_%v%v_%v", tw.User.Id, tw.User.ScreenName,tw.Id,captionText,filename )
		fnMap[filename] = longFilenames[i]
	}
	return fnMap
}

