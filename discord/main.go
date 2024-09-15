package discord

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Arno500/go-plex-client"
	discordRP "github.com/fawni/rp"
	rpc "github.com/fawni/rp/rpc"
	i18npkg "github.com/nicksnyder/go-i18n/v2/i18n"
	"gitlab.com/Arno500/plex-richpresence/i18n"
	"gitlab.com/Arno500/plex-richpresence/types"
)

var currentPlayState types.PlayState
var discord *rpc.Client
var client = &http.Client{
	Timeout: 5 * time.Second,
}

const EMPTY_THUMB_STRING = "plex"

// InitDiscordClient prepares Discord's RPC API to allow Rich Presence
func InitDiscordClient() {
	if discord == nil || !discord.Logged {
		discordInstance, err := discordRP.NewClient("803556010307616788")
		if err != nil {
			log.Println(err)
			return
		}
		discord = discordInstance
	}
}

// LogoutDiscordClient logout from Discord
func LogoutDiscordClient() {
	discord.Logout()
}

func getThumbnailLink(thumbKey string, plexInstance *plex.Plex) string {
	if currentPlayState.Thumb.PlexThumbUrl == thumbKey {
		if currentPlayState.Thumb.ImgLink != EMPTY_THUMB_STRING {
			return currentPlayState.Thumb.ImgLink
		}
	}
	currentPlayState.Thumb.PlexThumbUrl = thumbKey
	plexThumbLink := fmt.Sprintf("%s/photo/:/transcode?width=450&height=253&minSize=1&upscale=1&X-Plex-Token=%s&url=%s", plexInstance.URL, plexInstance.Token, thumbKey)

	thumbResp, err := client.Get(plexThumbLink)
	if err != nil {
		log.Printf("Couldn't get thumbnail from Plex (%s)", err)
		currentPlayState.Thumb.ImgLink = EMPTY_THUMB_STRING
		return EMPTY_THUMB_STRING
	}
	defer thumbResp.Body.Close()
	b, err := io.ReadAll(thumbResp.Body)
	if err != nil {
		log.Printf("Couldn't read thumbnail from Plex (%s)", err)
		currentPlayState.Thumb.ImgLink = EMPTY_THUMB_STRING
		return EMPTY_THUMB_STRING
	}

	imgUrl, err := UploadImage(b, thumbKey)
	if err != nil {
		log.Println("Error uploading image to litterbox: ", err)
		currentPlayState.Thumb.ImgLink = EMPTY_THUMB_STRING
		return EMPTY_THUMB_STRING
	}
	currentPlayState.Thumb.ImgLink = imgUrl
	return imgUrl
}

// SetRichPresence allows to send Rich Presence informations to Plex from a session info
func SetRichPresence(session types.PlexStableSession) {
	InitDiscordClient()
	now := time.Now()
	currentPlayState.Alteration.Item = false
	currentPlayState.Alteration.Time = false
	activityInfos := rpc.Activity{
		LargeImage: "plex",
		LargeText:  "Plex",
	}
	if session.Media.Type == "track" {
		activityInfos.Type = rpc.ActivityTypeListening
	} else {
		activityInfos.Type = rpc.ActivityTypeWatching
	}
	if currentPlayState.PlayingItem == nil || currentPlayState.PlayingItem.Media.GUID.String() != session.Media.GUID.String() {
		currentPlayState.PlayingItem = &session
		currentPlayState.Alteration.Item = true
	}
	if currentPlayState.PlayState != session.Session.State {
		currentPlayState.PlayState = session.Session.State
		currentPlayState.Alteration.Time = true
	}
	if session.Session.State == "paused" {
		if session.Media.Type == "track" {
			activityInfos.SmallImage = "pause"
			activityInfos.SmallText = i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
				DefaultMessage: &i18npkg.Message{
					ID:    "Paused",
					Other: "Paused",
				},
			})
		}
	} else if (session.Session.State == "playing" || session.Session.State == "buffering") && session.Media.Type != "photo" {
		if session.Media.Type == "track" {
			activityInfos.SmallImage = "play"
			activityInfos.SmallText = i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
				DefaultMessage: &i18npkg.Message{
					ID:    "NowPlaying",
					Other: "Playing",
				},
			})
		}
		if session.Session.State == "playing" {
			timeResetThreshold, _ := time.ParseDuration("4s")
			progress, _ := time.ParseDuration(strconv.FormatInt(session.Session.ViewOffset/1000, 10) + "s")
			calculatedStartTime := now.Add(-progress)
			duration, _ := time.ParseDuration(strconv.FormatInt(session.Media.Duration, 10) + "ms")
			calculatedEndTime := calculatedStartTime.Add(duration)
			activityInfos.Timestamps = &rpc.Timestamps{
				Start: &calculatedStartTime,
				End:   &calculatedEndTime,
			}
			if currentPlayState.LastCalculatedTime.Sub(calculatedEndTime).Abs() > timeResetThreshold {
				log.Printf("A seek or a media change was detected, updating state...")
				currentPlayState.Alteration.Time = true
				currentPlayState.LastCalculatedTime = calculatedEndTime
			}
		} else {
			currentPlayState.Alteration.Time = true
		}
	} else if session.Media.Type == "photo" {
		activityInfos.SmallImage = "camera"
	} else {
		log.Printf("Nothing is playing, closing connection to Discord.")
		LogoutDiscordClient()
		return
	}

	if currentPlayState.Alteration.Item || currentPlayState.Alteration.Time {
		if session.Media.Type == "episode" {
			// Episode title
			activityInfos.State = fmt.Sprintf("%s", session.Media.Title)
			// Show title
			activityInfos.Details = session.Media.GrandparentTitle
			activityInfos.LargeImage = getThumbnailLink(session.Media.GrandparentThumbnail, session.PlexInstance)
			activityInfos.LargeText = fmt.Sprintf("Season %02d, Episode %02d", session.Media.ParentIndex, session.Media.Index)
			activityInfos.Buttons = append(activityInfos.Buttons, &rpc.Button{
				Label: i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
					DefaultMessage: &i18npkg.Message{
						ID:    "ShowDetails",
						Other: "Show details on Plex",
					},
				}),
				Url: fmt.Sprintf("https://app.plex.tv/desktop/#!/provider/tv.plex.provider.discover/details?key=/library/metadata/%s", path.Base(session.Media.GrandparentGUID.EscapedPath())),
			})
		} else if session.Media.Type == "movie" {
			var formattedMovieName string
			if session.Media.Year > 0 {
				formattedMovieName = fmt.Sprintf("%s (%d)", session.Media.Title, session.Media.Year)
			} else {
				formattedMovieName = session.Media.Title
			}
			// Movie Director(s)
			if len(session.Media.Director) > 0 {
				directors := make([]string, len(session.Media.Director))
				for i, director := range session.Media.Director {
					directors[i] = director.Tag
				}
				activityInfos.State = strings.Join(directors, ", ")
			} else {
				activityInfos.State = "(⌐■_■)"
			}
			activityInfos.Details = formattedMovieName
			activityInfos.LargeImage = getThumbnailLink(session.Media.Thumbnail, session.PlexInstance)
			activityInfos.Buttons = append(activityInfos.Buttons, &rpc.Button{
				Label: i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
					DefaultMessage: &i18npkg.Message{
						ID:    "MovieDetails",
						Other: "Movie details on Plex",
					},
				}),
				Url: fmt.Sprintf("https://app.plex.tv/desktop/#!/provider/tv.plex.provider.discover/details?key=/library/metadata/%s", path.Base(session.Media.GUID.EscapedPath())),
			})
		} else if session.Media.Type == "track" {
			artist := ""
			if session.Media.OriginalTitle != "" {
				artist = session.Media.OriginalTitle
			} else {
				artist = session.Media.GrandparentTitle
			}
			activityInfos.State = fmt.Sprintf("by %s", artist)
			activityInfos.LargeImage = getThumbnailLink(session.Media.ParentThumbnail, session.PlexInstance)
			activityInfos.Buttons = append(activityInfos.Buttons, &rpc.Button{
				Label: i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
					DefaultMessage: &i18npkg.Message{
						ID:    "TrackDetails",
						Other: "Track details on Plex",
					},
				}),
				Url: fmt.Sprintf("https://listen.plex.tv/track/%s?parentGuid=%s&grandparentGuid=%s", path.Base(session.Media.GUID.EscapedPath()), path.Base(session.Media.ParentGUID.EscapedPath()), path.Base(session.Media.GrandparentGUID.EscapedPath())),
			}, &rpc.Button{
				Label: i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
					DefaultMessage: &i18npkg.Message{
						ID:    "YoutubeSearch",
						Other: "Search on YouTube",
					},
				}),
				Url: fmt.Sprintf("https://www.youtube.com/results?search_query=%s", url.QueryEscape(artist+" "+session.Media.Title)),
			})
			activityInfos.Details = session.Media.Title
			activityInfos.LargeText = fmt.Sprintf("on %s", session.Media.ParentTitle)
		} else if session.Media.Type == "photo" {
			text := i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
				DefaultMessage: &i18npkg.Message{
					ID:    "WatchingPhotos",
					Other: "Watching photos",
				},
			})
			activityInfos.State = text
			activityInfos.SmallText = text
			activityInfos.Details = session.Media.Title
		} else if session.Media.Type == "clip" {
			// Trailer data (preroll)
			activityInfos.State = session.Media.Title
			activityInfos.SmallText = "Preroll"
		}
		err := discord.SetActivity(&activityInfos)
		if err != nil {
			log.Printf("An error occured when setting the activity in Discord: %v", err)
			discord = nil
		} else {
			log.Printf("Discord activity set")
		}
	}
}
