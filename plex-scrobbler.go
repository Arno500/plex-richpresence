package main

import (
	"log"
	_ "net/http/pprof"
	"os"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/jrudio/go-plex-client"

	"gitlab.com/Arno500/plex-scrobbler/icon"
)

func main() {
	log.SetOutput(os.Stdout)
	go mainFunc()
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("Plex-Scrobbler")
	systray.SetTooltip("Plex-Scrobbler")
	quitChan := systray.AddMenuItem("Quitter", "Ferme l'application et désactive le scrobbling").ClickedCh
	for {
		_, ok := <-quitChan
		if ok == true {
			systray.Quit()
		} else {
			break
		}
	}
}

func mainFunc() {
	sum := 0
	var Plex *plex.Plex
	var serversURI []string
	for {
		if sum%50 == 0 {
			Plex = GetPlexTv()
			servers, err := Plex.GetServers()
			if err != nil {
				return
			}
			var wg sync.WaitGroup

			serversURI = nil
			for _, server := range servers {
				wg.Add(1)
				go GetGoodURI(server, &serversURI, &wg)
			}

			wg.Wait()
		}

		foundSession := false

		for _, serverURI := range serversURI {
			sessions := GetMySessionsFromServer(serverURI)
			if len(sessions) >= 1 {
				SetRichPresence(sessions[0])
				foundSession = true
				break
			}
		}
		if !foundSession {
			LogoutDiscordClient()
		}
		time.Sleep(3 * time.Second)
		sum++
	}
}

func onExit() {
	LogoutDiscordClient()
	os.Exit(0)
}
