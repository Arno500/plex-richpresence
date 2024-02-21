package plex

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"slices"
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
	reversed := slices.Clone(server.Connection)
	// After some reflection, I realized that most people would want external access, so this improve the chances of finding a working address quickly
	// + we don't need any speed anyway, so even if we get a relay, it's fine
	slices.Reverse(reversed)
	for _, uri := range reversed {
		Plex := GetPlex(uri.URI, server.AccessToken)
		log.Printf("%s: Trying to connect to %s", server.Name, uri.URI)
		_, err := Plex.GetLibraries()
		if err == nil {
			log.Printf("%s: %s was successfully contacted", server.Name, uri.URI)
			if uri.Relay {
				log.Printf("%s: This is a relay, so we should have correct access anyway", server.Name)
			}
			return uri, true
		}
	}

	log.Printf("Couldn't find any working address for server %s", server.Name)
	return plex.Connection{}, false
}

// AreSameUser checks if user info between the server and PlexTV are matching
func AreSameUser(user1 plex.User, user2 plex.UserPlexTV) bool {
	return user1.Title == user2.Title || user1.Email == user2.Email || user1.Username == user2.Username
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
			Live:				  session.Live == "1",
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

func refreshMetadata(session *types.PlexStableSession, Plex *plex.Plex) {
	mediaInfos, _ := Plex.GetMetadata(session.Media.RatingKey)
	guid, _ := url.Parse(mediaInfos.MediaContainer.Metadata[0].GUID)
	parentGuid, _ := url.Parse(mediaInfos.MediaContainer.Metadata[0].ParentGUID)
	grandparentGuid, _ := url.Parse(mediaInfos.MediaContainer.Metadata[0].GrandparentGUID)
	session.Media = types.PlexMediaKey{
			RatingKey:            mediaInfos.MediaContainer.Metadata[0].RatingKey,
			Type:                 mediaInfos.MediaContainer.Metadata[0].Type,
			Live:				  mediaInfos.MediaContainer.Metadata[0].Live == "1",
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
		}
}

// StartWebsocketConnections starts a WebSocket connection to a server, and manages events from them
func StartWebsocketConnections(server plex.PMSDevices, accountData plex.UserPlexTV, runningSockets *map[string]*chan interface{}, reconnectionChannelTimer chan bool) {
	Plex := GetPlex(server.Connection[0].URI, server.AccessToken)

	cancelChan := make(chan interface{})

	onConnectionClose := func() {
		log.Printf("Disconnected from %s", server.Name)
		delete(*runningSockets, server.ClientIdentifier)
		select {
			case reconnectionChannelTimer <- true:
			default:
		}
	}

	onError := func(err error) {
		log.Printf("Couldn't connect or lost connection to %s", server.Name)
		log.Println(err)
		select {
			case cancelChan <- true:
			default:
		}
		onConnectionClose()
	}

	events := plex.NewNotificationEvents()
	events.OnPlaying(func(n plex.NotificationContainer) {
		owned, _ := strconv.ParseBool(server.Owned)
		var stableSession types.PlexStableSession
		notif := n.PlaySessionStateNotification[0]
		log.Printf("Playing event received: %+v", notif)
		if owned {
			cacheEntry, entryExists := sessionCache[notif.SessionKey]
			log.Printf("Server %s is owned by the user, checking if the session is already cached", server.Name)
			if entryExists && cacheEntry.Media.Live {
				log.Printf("Session was in the cache, but this is live, we want to make sure that the session is not stale")
				refreshMetadata(&cacheEntry, Plex)
			}
			if entryExists && cacheEntry.Media.RatingKey == notif.RatingKey {
				log.Printf("Session was in the cache, updating state and progress")
				cacheEntry.Session.State = notif.State
				cacheEntry.Session.ViewOffset = notif.ViewOffset
				stableSession = cacheEntry
			} else {
				log.Printf("Session was not in the cache, retrieving data")
				sessions, err := Plex.GetSessions()
				if err != nil {
					onError(err)
				}
				for _, session := range sessions.MediaContainer.Metadata {
					if notif.SessionKey == session.SessionKey && session.User.ID == "1" {
						stableSession = createSessionFromSessionObject(notif, session, Plex)
						sessionCache[notif.SessionKey] = stableSession
						break
					}
				}
			}
		} else {
			log.Printf("Server %s is not owned by the user, using notification as is", server.Name)
			stableSession = createSessionFromWSNotif(n.PlaySessionStateNotification[0], Plex)
		}
		if stableSession.Session.State != "" {
			if !MachineIsEnabled(stableSession.Player) {
				log.Printf("Player %s is not enabled, ignoring", stableSession.Player.Title)
				return
			}
			discord.SetRichPresence(stableSession)
			if stableSession.Session.State == "stopped" {
				delete(sessionCache, notif.RatingKey)
			}
		}
	})

	Plex.SubscribeToNotifications(events, cancelChan, onError, onConnectionClose)
	log.Printf("Plex Rich Presence is now receiving notifications from %s (%s)", server.Name, server.ProductVersion)
	(*runningSockets)[server.ClientIdentifier] = &cancelChan
}

func StartConnectThread(targetServer *plex.PMSDevices, accountData plex.UserPlexTV, runningSockets *map[string]*chan interface{}, reconnectionChannelTimer chan bool) {
	if _, ok := (*runningSockets)[targetServer.ClientIdentifier]; !ok {
		goodConnection, found := GetGoodURI(targetServer)
		if !found {
			return
		}
		targetServer.Connection = []plex.Connection{goodConnection}
		StartWebsocketConnections(*targetServer, accountData, runningSockets, reconnectionChannelTimer)
	}
}
