package main

import (
	"time"

	"github.com/Arno500/go-plex-client"
)

// PlayState contains the last play state to only refresh time when needed
type PlayState struct {
	LastCalculatedTime time.Time
	PlayState          string
	DiscordConnected   bool
	PlayingItem        PlexStableSession
}

// PinSettings corresponds to the stored pin data from Plex
type PinSettings struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

// PlexStableSession is the object we send to the function that send the Rich Presence
type PlexStableSession struct {
	Media   PlexMediaKey
	Session PlexSessionKey
}

// PlexMediaKey is a subkey of PlexStableSession
type PlexMediaKey struct {
	RatingKey        string
	Type             string
	Duration         int64
	Director         []plex.TaggedData
	Index            int64
	ParentIndex      int64
	GrandparentTitle string
	OriginalTitle    string
	ParentTitle      string
	Title            string
	Year             int
}

// PlexSessionKey is a subkey of PlexStableSession
type PlexSessionKey struct {
	State      string
	ViewOffset int64
}

// PlexRPSettings is the stored structure on the file system
type PlexRPSettings struct {
	TimeMode         string      `json:"timeMode" default:"elapsed"`
	ClientIdentifier string      `json:"clientId"`
	AccessToken      string      `json:"accessToken"`
	Pin              PinSettings `json:"pin"`
}
