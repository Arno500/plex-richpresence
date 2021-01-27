package main

import (
	"fmt"
	"log"
	"time"

	discord "github.com/hugolgst/rich-go/client"
	"github.com/jrudio/go-plex-client"
)

var currentPlayState PlayState

//InitDiscordClient prepares Discord's RPC API to allow Rich Presence
func InitDiscordClient() {
	if currentPlayState.DiscordConnected {
		return
	}
	err := discord.Login("803556010307616788")
	if err != nil {
		panic(err)
	}
	currentPlayState.DiscordConnected = true
}

// LogoutDiscordClient logout from Discord
func LogoutDiscordClient() {
	if currentPlayState.DiscordConnected {
		discord.Logout()
		currentPlayState.DiscordConnected = false
	}
}

// SetRichPresence allows to send Rich Presence informations to Plex from a session info
func SetRichPresence(session plex.MetadataV1) {
	now := time.Now()
	stateAltered := false
	var activityInfos discord.Activity
	if currentPlayState.CurrentlyPlaying != session.Metadata.GUID {
		currentPlayState.CurrentlyPlaying = session.Metadata.GUID
		stateAltered = true
	}
	if currentPlayState.Paused != (session.Metadata.Player.State == "paused") {
		currentPlayState.Paused = session.Metadata.Player.State == "paused"
		stateAltered = true
	}
	// TODO: i18n
	if session.Metadata.Player.State == "paused" {
		activityInfos = discord.Activity{
			LargeImage: "plex",
			LargeText:  "Plex",
			SmallImage: "pause",
			SmallText:  "En pause",
		}
	} else if session.Metadata.Player.State == "playing" {
		timeResetThreshold, _ := time.ParseDuration("8s")
		duration, _ := time.ParseDuration(session.ViewOffset + "ms")
		calculatedStartTime := now.Add(-duration)
		if !(currentPlayState.LastCalculatedStart.Add(-timeResetThreshold).Before(calculatedStartTime) && currentPlayState.LastCalculatedStart.Add(timeResetThreshold).After(calculatedStartTime)) {
			log.Printf("A seeking or a media change was detected, adjusting")
			stateAltered = true
			currentPlayState.LastCalculatedStart = calculatedStartTime
		}
		activityInfos = discord.Activity{
			LargeImage: "plex",
			LargeText:  "Plex",
			SmallImage: "play",
			SmallText:  "En cours de lecture",
			Timestamps: &discord.Timestamps{
				Start: &calculatedStartTime,
			},
		}
	} else {
		discord.Logout()
		return
	}
	if session.Type == "episode" {
		// Season - Ep
		activityInfos.State = fmt.Sprintf("Saison %d, épisode %d", session.ParentIndex, session.Index)
		// Show
		activityInfos.Details = session.GrandparentTitle
	} else if session.Type == "movie" {
		// Movie Director
		activityInfos.State = session.Director[0].Tag
		// Movie title
		activityInfos.Details = session.Title
	}
	if stateAltered {
		InitDiscordClient()
		_ = discord.SetActivity(activityInfos)
		log.Printf("Discord activity set")
	}
}
