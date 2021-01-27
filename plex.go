package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/jrudio/go-plex-client"
)

// GetPlexTv instance
func GetPlexTv() *plex.Plex {
	err := CheckToken()
	var Plex = plex.Plex{
		ClientIdentifier: storedSettings.ClientIdentifier,
		Token:            storedSettings.AccessToken,
		HTTPClient: http.Client{
			Timeout: 3 * time.Second,
		},
	}

	Plex.Headers.Product = AppName
	Plex.Headers.Platform = runtime.GOOS
	Plex.Headers.PlatformVersion = "0.0.0"
	Plex.Headers.Version = "0.0.1"
	Plex.Headers.Accept = "application/json"
	Plex.Headers.ContentType = "application/json"
	Plex.Headers.ContainerSize = "Plex-Container-Size=50"
	Plex.Headers.ContainerStart = "X-Plex-Container-Start=0"

	name, err := os.Hostname()
	if err != nil {
		Plex.Headers.Device = name
	} else {
		Plex.Headers.Device = "Arno & Co"
	}
	return &Plex
}

// GetPlex instance
func GetPlex(instance string) *plex.Plex {
	Plex := GetPlexTv()
	Plex.URL = instance
	Plex.Headers.ClientIdentifier = storedSettings.ClientIdentifier
	Plex.Headers.Token = storedSettings.AccessToken
	return Plex
}

// GetGoodURI finds the working URL for a working server
func GetGoodURI(server plex.PMSDevices, serversURI *[]string, wg *sync.WaitGroup) {
	defer wg.Done()

	found := false

	for _, uri := range server.Connection {
		parsedURL, _ := url.Parse(uri.URI)
		log.Printf("%s: Trying to connect to %s", server.Name, parsedURL.Host)
		conn, _ := net.DialTimeout("tcp", parsedURL.Host, 400*time.Millisecond)
		if conn != nil {
			log.Printf("%s: %s was successfully contacted", server.Name, parsedURL.Host)
			*serversURI = append(*serversURI, uri.URI)
			found = true
			break
		}
	}
	if !found {
		log.Printf("Couldn't find any working address for server %s", server.Name)
	}
	return
}

//GetMySessionsFromServer :/
func GetMySessionsFromServer(serverURL string) []plex.MetadataV1 {
	Plex := GetPlex(serverURL)
	// TODO: Only get sessions from the user itself, not every sessions on the server
	sessions, _ := Plex.GetSessions()
	// for _, session := range sessions.MediaContainer.Metadata {
	// 	fmt.Printf("%+v", session)
	// }
	return sessions.MediaContainer.Metadata
}
