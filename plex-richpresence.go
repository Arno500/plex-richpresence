package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	plexpkg "github.com/Arno500/go-plex-client"

	"gitlab.com/Arno500/plex-richpresence/autoupdate"
	"gitlab.com/Arno500/plex-richpresence/discord"
	"gitlab.com/Arno500/plex-richpresence/gui"
	"gitlab.com/Arno500/plex-richpresence/i18n"
	"gitlab.com/Arno500/plex-richpresence/plex"
	"gitlab.com/Arno500/plex-richpresence/settings"
	"gitlab.com/Arno500/plex-richpresence/types"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("An error occured when contacting server, the program will now exit: %s", r)
		}
	}()
	log.SetOutput(io.MultiWriter(&lumberjack.Logger{
		Filename:   filepath.Join(settings.ConfigFolders[0].Path, "debug_log.txt"),
		MaxSize:    5,
		MaxBackups: 3,
		MaxAge:     5,
		Compress:   true,
	}, os.Stdout))
	gui.StartTray(onReady, onExit)
}

func onReady() {
	i18n.InitLocale()
	settings.Load()
	gui.SetupTray()
	autoupdate.Autoupdate()
	ctx, cancelMain := context.WithCancel(context.Background())
	go mainFunc(ctx)
	defer func() {
		if r := recover(); r != nil {
			log.Println("System error, retrying in 10 seconds", r)
			time.Sleep(5 * time.Second)
			mainFunc(ctx)
		}
	}()
	for {
		select {
		case <-gui.TrayHandlers.TimeElapsed.ClickedCh:
			gui.ToggleTimeMode(gui.TrayHandlers.TimeElapsed, gui.TrayHandlers.TimeRemaining, "elapsed")
		case <-gui.TrayHandlers.TimeRemaining.ClickedCh:
			gui.ToggleTimeMode(gui.TrayHandlers.TimeRemaining, gui.TrayHandlers.TimeElapsed, "remaining")
		case <-gui.TrayHandlers.EnabledDeviceByDefault.ClickedCh:
			gui.ToggleAutoEnableDevices(gui.TrayHandlers.EnabledDeviceByDefault)
		case <-gui.TrayHandlers.AutoLaunch.ClickedCh:
			gui.ToggleAutoStart(gui.TrayHandlers.AutoLaunch)
		case <-gui.TrayHandlers.DisconnectBtn.ClickedCh:
			settings.StoredSettings.AccessToken = ""
			settings.StoredSettings.Pin = types.PinSettings{}
			settings.Save()
			cancelMain()
			go mainFunc(ctx)
		case <-gui.TrayHandlers.QuitBtn.ClickedCh:
			gui.Quit()
		}
	}
}

func disconnectSockets(sockets *map[string]*chan interface{}) {
	for _, socket := range *sockets {
		select {
		case *socket <- true:
		default:
		}
	}
	*sockets = make(map[string]*chan interface{})
}

func mainFunc(ctx context.Context) {
	var Plex *plexpkg.Plex
	var filteredServers []plexpkg.PMSDevices
	var accountData plexpkg.UserPlexTV
	runningSockets := make(map[string]*chan interface{})

	timeoutchan := make(chan bool)

	// TODO: Need to seed first session data if admin (Plex doesn't resend data)

	for {
		var wg sync.WaitGroup
		log.Printf("Refreshing servers")
		Plex = plex.GetPlexTv()
		accountData, _ = Plex.MyAccount()
		servers, _ := Plex.GetServers()

		filteredServers = nil
		for _, server := range servers {
			if _, ok := runningSockets[server.ClientIdentifier]; !ok {
				wg.Add(1)
				go plex.GetGoodURI(server, &filteredServers, &wg)
			}
		}

		wg.Wait()

		for _, server := range filteredServers {
			go plex.StartWebsocketConnections(server, &accountData, &runningSockets)
			log.Printf("Sucessfully connected to %s WebSocket", server.Connection[0].URI)
		}

		// Basically wait 60 seconds in another thread, then finish the loop iteration to scan servers again (thus refreshing everything)
		go func() {
			<-time.After(60 * time.Second)
			timeoutchan <- true
		}()

		select {
		case <-ctx.Done():
			disconnectSockets(&runningSockets)
			return
		case <-timeoutchan:
		}
	}
}

func onExit() {
	discord.LogoutDiscordClient()
	os.Exit(0)
}
