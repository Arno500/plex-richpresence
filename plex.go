package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/Arno500/go-plex-client"
)

func setupHeaders(Plex *plex.Plex) {
	Plex.Headers.Product = AppName
	Plex.Headers.Platform = runtime.GOOS
	Plex.Headers.PlatformVersion = "0.0.1"
	Plex.Headers.Version = "0.0.1"
	Plex.Headers.Accept = "application/json"
	Plex.Headers.ContentType = "application/json"
	Plex.Headers.ContainerSize = "Plex-Container-Size=50"
	Plex.Headers.ContainerStart = "X-Plex-Container-Start=0"

	name, err := os.Hostname()
	if err == nil {
		Plex.Headers.Device = name
	} else {
		Plex.Headers.Device = "Arno & Co"
	}
}

// GetPlexTv instance
func GetPlexTv() *plex.Plex {
	err := CheckToken()
	if err != nil {
		log.Fatalln(err)
	}
	var Plex = plex.Plex{
		ClientIdentifier: StoredSettings.ClientIdentifier,
		Token:            StoredSettings.AccessToken,
		HTTPClient: http.Client{
			Timeout: 3 * time.Second,
		},
	}

	setupHeaders(&Plex)

	return &Plex
}

// GetPlex instance
func GetPlex(instance string, token string) *plex.Plex {
	Plex := GetPlexTv()
	Plex.URL = instance
	Plex.Headers.ClientIdentifier = StoredSettings.ClientIdentifier
	Plex.Token = token
	Plex.Headers.Token = token
	return Plex
}

// GetGoodURI finds the working URL for a working server
func GetGoodURI(server plex.PMSDevices, destinationSlice *[]plex.PMSDevices, wg *sync.WaitGroup) {
	defer wg.Done()

	found := false

	for _, uri := range server.Connection {
		parsedURL, _ := url.Parse(uri.URI)
		log.Printf("%s: Trying to connect to %s", server.Name, parsedURL.Host)
		conn, _ := net.DialTimeout("tcp", parsedURL.Host, 400*time.Millisecond)
		if conn != nil {
			log.Printf("%s: %s was successfully contacted", server.Name, parsedURL.Host)
			server.Connection = nil
			server.Connection = append(server.Connection, uri)
			*destinationSlice = append(*destinationSlice, server)
			found = true
			break
		}
	}
	if !found {
		log.Printf("Couldn't find any working address for server %s", server.Name)
	}
	return
}

var sessionCache = make(map[string]PlexStableSession)
var mediaCache = make(map[string]plex.MediaMetadata)

func createSessionFromWSNotif(wsNotif plex.PlaySessionStateNotification, Plex *plex.Plex) PlexStableSession {
	mediaInfos, entryExists := mediaCache[wsNotif.RatingKey]
	if !entryExists {
		mediaInfos, _ = Plex.GetMetadata(wsNotif.RatingKey)
		mediaCache[wsNotif.RatingKey] = mediaInfos
	}
	return PlexStableSession{
		Media: PlexMediaKey{
			RatingKey:        wsNotif.RatingKey,
			Type:             mediaInfos.MediaContainer.Metadata[0].Type,
			Duration:         mediaInfos.MediaContainer.Metadata[0].Duration,
			Index:            mediaInfos.MediaContainer.Metadata[0].Index,
			ParentIndex:      mediaInfos.MediaContainer.Metadata[0].ParentIndex,
			Director:         mediaInfos.MediaContainer.Metadata[0].Director,
			GrandparentTitle: mediaInfos.MediaContainer.Metadata[0].GrandparentTitle,
			ParentTitle:      mediaInfos.MediaContainer.Metadata[0].ParentTitle,
			Title:            mediaInfos.MediaContainer.Metadata[0].Title,
			Year:             mediaInfos.MediaContainer.Metadata[0].Year,
		},
		Session: PlexSessionKey{
			State:      wsNotif.State,
			ViewOffset: wsNotif.ViewOffset,
		},
	}
}
func createSessionFromSessionObject(wsNotif plex.PlaySessionStateNotification, session plex.MetadataV1) PlexStableSession {
	year, _ := strconv.Atoi(session.Year)
	duration, _ := strconv.Atoi(session.Duration)
	return PlexStableSession{
		Media: PlexMediaKey{
			RatingKey:        session.RatingKey,
			Type:             session.Type,
			Duration:         duration,
			Index:            session.Index,
			ParentIndex:      session.ParentIndex,
			Director:         session.Director,
			GrandparentTitle: session.GrandparentTitle,
			ParentTitle:      session.ParentTitle,
			Title:            session.Title,
			Year:             year,
		},
		Session: PlexSessionKey{
			State:      wsNotif.State,
			ViewOffset: wsNotif.ViewOffset,
		},
	}
}

//StartWebsocketConnections starts a WebSocket connection to a server, and managing events from them/
func StartWebsocketConnections(server plex.PMSDevices, accountData *plex.UserPlexTV, runningSockets *[]*chan interface{}) {
	Plex := GetPlex(server.Connection[0].URI, server.AccessToken)

	cancelChan := make(chan interface{})

	onError := func(err error) {
		time.Sleep(5 * time.Second)
		StartWebsocketConnections(server, accountData, runningSockets)
	}

	events := plex.NewNotificationEvents()
	events.OnPlaying(func(n plex.NotificationContainer) {
		// GUID:""
		// Key:"/library/metadata/75477"
		// PlayQueueItemID:228823
		// RatingKey:"75477"
		// SessionKey:"3"
		// State:"playing"
		// URL:""
		// ViewOffset:0
		// TranscodeSession:"e3qilez0mo93hsyc0c5ummqv"
		owned, _ := strconv.ParseBool(server.Owned)
		var stableSession PlexStableSession
		notif := n.PlaySessionStateNotification[0]
		if owned {
			cacheEntry, entryExists := sessionCache[notif.RatingKey]
			if entryExists {
				cacheEntry.Session.State = notif.State
				cacheEntry.Session.ViewOffset = notif.ViewOffset
				stableSession = cacheEntry
			} else {
				sessions, err := Plex.GetSessions()
				if err != nil {
					log.Panic(err)
				}
				for _, session := range sessions.MediaContainer.Metadata {
					if notif.SessionKey == session.SessionKey && session.User.Title == accountData.Title {
						stableSession = createSessionFromSessionObject(notif, session)
						sessionCache[notif.RatingKey] = stableSession
						break
					}
				}
			}
		} else {
			stableSession = createSessionFromWSNotif(n.PlaySessionStateNotification[0], Plex)
		}
		if stableSession.Session.State != "" {
			SetRichPresence(stableSession, Plex, owned)
			if stableSession.Session.State == "stopped" {
				delete(sessionCache, notif.RatingKey)
			}
		}
	})

	Plex.SubscribeToNotifications(events, cancelChan, onError)
	*runningSockets = append(*runningSockets, &cancelChan)
}
