package main

import "time"

// PlayState contains the last play state to only refresh time when needed
type PlayState struct {
	LastCalculatedStart time.Time
	CurrentlyPlaying    string
	Paused              bool
	DiscordConnected    bool
}
