package types

import (
	"net/url"
	"time"

	"github.com/Arno500/go-plex-client"
	"github.com/nekr0z/systray"
)

type TrayHandlersStruct struct {
	TimeElapsed            *systray.MenuItem
	TimeRemaining          *systray.MenuItem
	EnabledDeviceByDefault *systray.MenuItem
	Devices                *systray.MenuItem
	AutoLaunch             *systray.MenuItem
	DisconnectBtn          *systray.MenuItem
	QuitBtn                *systray.MenuItem
}

// PlayState contains the last play state to only refresh time when needed
type PlayState struct {
	LastCalculatedTime time.Time
	PlayState          string
	PlayingItem        *PlexStableSession
	Thumb              struct {
		PlexThumbUrl string
		ImgLink      string
	}
	Alteration struct {
		Item bool
		Time bool
	}
}

// PlexStableSession is the object we send to the function that send the Rich Presence
type PlexStableSession struct {
	Media        PlexMediaKey
	Session      PlexSessionKey
	Player       PlexPlayerKey
	PlexInstance *plex.Plex
}

// PlexMediaKey is a subkey of PlexStableSession
type PlexMediaKey struct {
	RatingKey            string
	GUID                 url.URL
	ParentGUID           url.URL
	GrandparentGUID      url.URL
	Type                 string
	Duration             int64
	Director             []plex.TaggedData
	Thumbnail            string
	ParentThumbnail      string
	GrandparentThumbnail string
	Index                int64
	ParentIndex          int64
	GrandparentTitle     string
	OriginalTitle        string
	ParentTitle          string
	Title                string
	Year                 int
	Live                 bool
}

// PlexSessionKey is a subkey of PlexStableSession
type PlexSessionKey struct {
	State      string
	ViewOffset int64
}

// PlexPlayerKey contains the specific informations about the player
type PlexPlayerKey struct {
	ClientIdentifier string
	Title            string
	Product          string
}

// PinSettings corresponds to the stored pin data from Plex
type PinSettings struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

type Device struct {
	Identifier string `json:"name"`
	Enabled    bool   `json:"enabled"`
	Product    string `json:"product"`
	Title      string `json:"title"`
}

// PlexRPSettings is the stored structure on the file system
type PlexRPSettings struct {
	ClientIdentifier          string      `json:"clientId"`
	AccessToken               string      `json:"accessToken"`
	Pin                       PinSettings `json:"pin"`
	EnableNewDevicesByDefault bool        `json:"enableNewDevicesByDefault"`
	Devices                   []Device    `json:"selectedDevices"`
}

type DeviceMenuItem struct {
	Device   *Device
	MenuItem *systray.MenuItem
}
