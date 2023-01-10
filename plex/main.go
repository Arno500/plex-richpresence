package plex

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/Arno500/go-plex-client"
	"gitlab.com/Arno500/plex-richpresence/discord"
	"gitlab.com/Arno500/plex-richpresence/settings"
	"gitlab.com/Arno500/plex-richpresence/types"
)

// AppName contains the name of the application sent to Plex
var appName = "Plex Rich Presence by Arno & Co"

func setupHeaders(Plex *plex.Plex) {
	Plex.Headers.Product = appName
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
		log.Printf("Couldn't get or check the token, retrying in 10 seconds (%s)", err)
		time.Sleep(10 * time.Second)
		return GetPlexTv()
	}
	var Plex = plex.Plex{
		ClientIdentifier: settings.StoredSettings.ClientIdentifier,
		Token:            settings.StoredSettings.AccessToken,
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
	Plex.Headers.ClientIdentifier = settings.StoredSettings.ClientIdentifier
	Plex.Token = token
	Plex.Headers.Token = token
	return Plex
}

// GetGoodURI finds the working URL for a working server
func GetGoodURI(server *plex.PMSDevices) (plex.Connection, bool) {
	for _, uri := range server.Connection {
		parsedURL, _ := url.Parse(uri.URI)
		log.Printf("%s: Trying to connect to %s", server.Name, parsedURL.Host)
		conn, _ := net.DialTimeout("tcp", parsedURL.Host, 800*time.Millisecond)
		if conn != nil {
			log.Printf("%s: %s was successfully contacted", server.Name, parsedURL.Host)
			if uri.Relay {
				log.Printf("%s: This is a relay, so we should have correct access anyway", server.Name)
			}
			return uri, true
		}
	}

	log.Printf("Couldn't find any working address for server %s", server.Name)
	return plex.Connection{}, false
}

var sessionCache = make(map[string]types.PlexStableSession)
var mediaCache = make(map[string]plex.MediaMetadata)

func createSessionFromWSNotif(wsNotif plex.PlaySessionStateNotification, Plex *plex.Plex) types.PlexStableSession {
	mediaInfos, entryExists := mediaCache[wsNotif.RatingKey]
	if !entryExists {
		mediaInfos, _ = Plex.GetMetadata(wsNotif.RatingKey)
		mediaCache[wsNotif.RatingKey] = mediaInfos
	}
	var playerInfo types.PlexPlayerKey
	if mediaInfos.MediaContainer.Metadata[0].Player.MachineIdentifier != "" {
		playerInfo = types.PlexPlayerKey{
			ClientIdentifier: mediaInfos.MediaContainer.Metadata[0].Player.MachineIdentifier,
			Title:            mediaInfos.MediaContainer.Metadata[0].Player.Title,
			Product:          mediaInfos.MediaContainer.Metadata[0].Player.Product,
		}
	} else {
		device := GetDevice(Plex, wsNotif.ClientIdentifier)
		playerInfo = types.PlexPlayerKey{
			ClientIdentifier: device.ClientIdentifier,
			Title:            device.Name,
			Product:          device.Product,
		}
	}
	guid, _ := url.Parse(mediaInfos.MediaContainer.Metadata[0].GUID)
	parentGuid, _ := url.Parse(mediaInfos.MediaContainer.Metadata[0].ParentGUID)
	grandparentGuid, _ := url.Parse(mediaInfos.MediaContainer.Metadata[0].GrandparentGUID)
	return types.PlexStableSession{
		Media: types.PlexMediaKey{
			RatingKey:            wsNotif.RatingKey,
			Type:                 mediaInfos.MediaContainer.Metadata[0].Type,
			Duration:             int64(mediaInfos.MediaContainer.Metadata[0].Duration),
			Index:                mediaInfos.MediaContainer.Metadata[0].Index,
			ParentIndex:          mediaInfos.MediaContainer.Metadata[0].ParentIndex,
			Director:             mediaInfos.MediaContainer.Metadata[0].Director,
			GrandparentTitle:     mediaInfos.MediaContainer.Metadata[0].GrandparentTitle,
			OriginalTitle:        mediaInfos.MediaContainer.Metadata[0].OriginalTitle,
			ParentTitle:          mediaInfos.MediaContainer.Metadata[0].ParentTitle,
			Title:                mediaInfos.MediaContainer.Metadata[0].Title,
			Year:                 mediaInfos.MediaContainer.Metadata[0].Year,
			Thumbnail:            mediaInfos.MediaContainer.Metadata[0].Thumb,
			ParentThumbnail:      mediaInfos.MediaContainer.Metadata[0].ParentThumb,
			GrandparentThumbnail: mediaInfos.MediaContainer.Metadata[0].GrandparentThumb,
			GUID:                 *guid,
			ParentGUID:           *parentGuid,
			GrandparentGUID:      *grandparentGuid,
		},
		Session: types.PlexSessionKey{
			State:      wsNotif.State,
			ViewOffset: wsNotif.ViewOffset,
		},
		Player:       playerInfo,
		PlexInstance: Plex,
	}
}
func createSessionFromSessionObject(wsNotif plex.PlaySessionStateNotification, session plex.MetadataV1, Plex *plex.Plex) types.PlexStableSession {
	guid, _ := url.Parse(session.GUID)
	parentGuid, _ := url.Parse(session.ParentGUID)
	grandparentGuid, _ := url.Parse(session.GrandparentGUID)
	return types.PlexStableSession{
		Media: types.PlexMediaKey{
			RatingKey:            session.RatingKey,
			Type:                 session.Type,
			Duration:             session.Duration,
			Index:                session.Index,
			ParentIndex:          session.ParentIndex,
			Director:             session.Director,
			GrandparentTitle:     session.GrandparentTitle,
			OriginalTitle:        session.OriginalTitle,
			ParentTitle:          session.ParentTitle,
			Title:                session.Title,
			Year:                 session.Year,
			Thumbnail:            session.Metadata.Thumb,
			ParentThumbnail:      session.Metadata.ParentThumb,
			GrandparentThumbnail: session.Metadata.GrandparentThumb,
			GUID:                 *guid,
			ParentGUID:           *parentGuid,
			GrandparentGUID:      *grandparentGuid,
		},
		Session: types.PlexSessionKey{
			State:      wsNotif.State,
			ViewOffset: wsNotif.ViewOffset,
		},
		Player: types.PlexPlayerKey{
			ClientIdentifier: session.Player.MachineIdentifier,
			Title:            session.Player.Title,
			Product:          session.Player.Product,
		},
		PlexInstance: Plex,
	}
}

// StartWebsocketConnections starts a WebSocket connection to a server, and manages events from them
func StartWebsocketConnections(server plex.PMSDevices, accountData *plex.UserPlexTV, runningSockets *map[string]*chan interface{}) {
	Plex := GetPlex(server.Connection[0].URI, server.AccessToken)

	cancelChan := make(chan interface{})

	onConnectionClose := func() {
		delete(*runningSockets, server.ClientIdentifier)
	}

	onError := func(err error) {
		select {
		case cancelChan <- true:
		default:
		}
		log.Printf("Couldn't connect or lost connection to %s", server.Name)
		log.Println(err)
		onConnectionClose()
	}

	events := plex.NewNotificationEvents()
	events.OnPlaying(func(n plex.NotificationContainer) {
		owned, _ := strconv.ParseBool(server.Owned)
		var stableSession types.PlexStableSession
		notif := n.PlaySessionStateNotification[0]
		if owned {
			cacheEntry, entryExists := sessionCache[notif.SessionKey]
			if entryExists && cacheEntry.Media.RatingKey == notif.RatingKey {
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
						stableSession = createSessionFromSessionObject(notif, session, Plex)
						sessionCache[notif.SessionKey] = stableSession
						break
					}
				}
			}
		} else {
			stableSession = createSessionFromWSNotif(n.PlaySessionStateNotification[0], Plex)
		}
		if !MachineIsEnabled(stableSession.Player) {
			return
		}
		if stableSession.Session.State != "" {
			discord.SetRichPresence(stableSession, owned)
			if stableSession.Session.State == "stopped" {
				delete(sessionCache, notif.RatingKey)
			}
		}
	})

	Plex.SubscribeToNotifications(events, cancelChan, onError, onConnectionClose)
	(*runningSockets)[server.ClientIdentifier] = &cancelChan
}

func StartConnectThread(targetServer *plex.PMSDevices, accountData *plex.UserPlexTV, runningSockets *map[string]*chan interface{}) {
	if _, ok := (*runningSockets)[targetServer.ClientIdentifier]; !ok {
		goodConnection, found := GetGoodURI(targetServer)
		if !found {
			return
		}
		targetServer.Connection = []plex.Connection{goodConnection}
	}
	StartWebsocketConnections(*targetServer, accountData, runningSockets)
}
