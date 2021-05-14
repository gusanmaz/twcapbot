package twcapbot

import (
	"fmt"
	"github.com/gusanmaz/twigger"
)

const botName = "imgalt Bot"
const screenName = "kirmizigokyuzu"
var botAndScreenName = fmt.Sprintf("%v (%v)", botName, screenName)

var endNotes1 = fmt.Sprintf("Generated by %v.", botAndScreenName)
var endNotes2 = "The bot is currently at it's early beta stage."
var endNotes3 = "Use at your own risk! Feedbacks are appreciated 😇"

var endNotes  = fmt.Sprintf("%v %v %v", endNotes1, endNotes2, endNotes3)

var videoNote = fmt.Sprintf("%v cannot properly captionize video tweets for now.", botAndScreenName)
var gifNote   = fmt.Sprintf("%v cannot propery captionize tweets with GIF images for now", botAndScreenName)

var infoNoteTempl = "Tweet published at %v *** Tweet ID: %v *** User ID: %v"

func GetCaptionsForTweet(tw twigger.Tweet)[]string{
	captions := make([]string, 0)
	textNote := fmt.Sprintf("%v tweeted: %v", tw.Id, tw.FullText)
	captions = append(captions, textNote)
	infoNote := fmt.Sprintf(infoNoteTempl, tw.CreatedAt, tw.Id, tw.User.Id)
	captions = append(captions, infoNote)
	if tw.ContainsVideo(){
		captions = append(captions, videoNote)
	}
	if tw.ContainsGIF(){
		captions = append(captions, gifNote)
	}
	captions = append(captions, endNotes)
	return captions
}