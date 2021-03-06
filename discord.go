package main

import (
	// "fmt"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Arno500/go-plex-client"
	discord "github.com/hugolgst/rich-go/client"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var currentPlayState PlayState

//InitDiscordClient prepares Discord's RPC API to allow Rich Presence
func InitDiscordClient() {
	if currentPlayState.DiscordConnected {
		return
	}
	err := discord.Login("803556010307616788")
	if err != nil {
		log.Panicln(err)
	}
	currentPlayState.DiscordConnected = true
}

// LogoutDiscordClient logout from Discord
func LogoutDiscordClient() {
	if currentPlayState.DiscordConnected {
		currentPlayState.DiscordConnected = false
		discord.Logout()
	}
}

// SetRichPresence allows to send Rich Presence informations to Plex from a session info
func SetRichPresence(session PlexStableSession, plexInstance *plex.Plex, owned bool) {
	now := time.Now()
	stateAltered := false
	activityInfos := discord.Activity{
		LargeImage: "plex",
		LargeText:  "Plex",
	}
	if currentPlayState.PlayingItem.Media.RatingKey != session.Media.RatingKey {
		currentPlayState.PlayingItem = session
		stateAltered = true
	}
	if currentPlayState.PlayState != session.Session.State {
		currentPlayState.PlayState = session.Session.State
		stateAltered = true
	}
	if session.Session.State == "paused" {
		activityInfos.SmallImage = "pause"
		activityInfos.SmallText = Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Paused",
				Other: "Paused",
			},
		})
	} else if (session.Session.State == "playing" || session.Session.State == "buffering") && session.Media.Type != "photo" {
		activityInfos.SmallImage = "play"
		activityInfos.SmallText = Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "NowPlaying",
				Other: "Playing",
			},
		})
		if session.Session.State == "playing" {
			timeResetThreshold, _ := time.ParseDuration("4s")
			progress, _ := time.ParseDuration(strconv.FormatInt(session.Session.ViewOffset, 10) + "ms")
			if StoredSettings.TimeMode == "elapsed" {
				calculatedStartTime := now.Add(-progress)
				if !(currentPlayState.LastCalculatedTime.Add(-timeResetThreshold).Before(calculatedStartTime) && currentPlayState.LastCalculatedTime.Add(timeResetThreshold).After(calculatedStartTime)) {
					log.Printf("A seeking or a media change was detected, adjusting")
					stateAltered = true
					currentPlayState.LastCalculatedTime = calculatedStartTime
				}
				activityInfos.Timestamps = &discord.Timestamps{
					Start: &calculatedStartTime,
				}
			} else if StoredSettings.TimeMode == "remaining" {
				duration, _ := time.ParseDuration(strconv.Itoa(session.Media.Duration) + "ms")
				remaining := duration - progress
				calculatedEndTime := now.Add(remaining)
				if !(currentPlayState.LastCalculatedTime.Add(-timeResetThreshold).Before(calculatedEndTime) && currentPlayState.LastCalculatedTime.Add(timeResetThreshold).After(calculatedEndTime)) {
					log.Printf("A seeking or a media change was detected, adjusting")
					stateAltered = true
					currentPlayState.LastCalculatedTime = calculatedEndTime
				}
				activityInfos.Timestamps = &discord.Timestamps{
					Start: &calculatedEndTime,
					End:   &calculatedEndTime,
				}
			}
		}
	} else if session.Media.Type == "photo" {
		activityInfos.SmallImage = "camera"
	} else {
		log.Printf("Nothing is playing, closing connection to Discord.")
		LogoutDiscordClient()
		return
	}

	if stateAltered {
		if session.Media.Type == "episode" {
			// Season - Ep
			activityInfos.State = Localizer.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "SeasonEpisodeProgress",
					Other: "Season {{.Season}}, episode {{.Episode}}",
				},
				TemplateData: map[string]interface{}{
					"Season":  session.Media.ParentIndex,
					"Episode": session.Media.Index,
				},
			})
			// Show
			activityInfos.Details = session.Media.GrandparentTitle
		} else if session.Media.Type == "movie" {
			// Movie Director
			activityInfos.State = session.Media.Director[0].Tag
			// Movie title
			activityInfos.Details = fmt.Sprintf("%s (%s)", session.Media.Title, strconv.Itoa(session.Media.Year))
		} else if session.Media.Type == "track" {
			if session.Media.OriginalTitle != "" {
				activityInfos.State = session.Media.OriginalTitle
			} else {
				activityInfos.State = session.Media.GrandparentTitle
			}
			activityInfos.Details = fmt.Sprintf("%s (%s)", session.Media.Title, session.Media.ParentTitle)
		} else if session.Media.Type == "photo" {
			text := Localizer.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "WatchingPhotos",
					Other: "Watching photos",
				},
			})
			activityInfos.State = text
			activityInfos.SmallText = text
			activityInfos.Details = session.Media.Title
		}
		InitDiscordClient()
		err := discord.SetActivity(activityInfos)
		if err != nil {
			log.Printf("An error occured when setting the activity in Discord: %v", err)
		} else {
			log.Printf("Discord activity set")
		}
	}
}
