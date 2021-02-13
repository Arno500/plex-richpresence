package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Arno500/go-plex-client"
	"github.com/emersion/go-autostart"
	"github.com/getlantern/systray"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"gitlab.com/Arno500/plex-richpresence/icon"
	"gitlab.com/Arno500/plex-richpresence/settings"
)

// AppName contains the name of the application sent to Plex
var AppName = "Plex Rich Presence by Arno & Co"

// StoredSettings contains the global settings of the app
var StoredSettings = PlexRPSettings{
	TimeMode: "elapsed",
}

var appExec, _ = os.Executable()
var appExecResolved, _ = filepath.EvalSymlinks(appExec)

var appAutoStart = &autostart.App{
	Name:        "Plex Rich Presence",
	DisplayName: "Plex Rich Presence",
	Exec:        []string{appExecResolved},
}

func main() {
	f, err := settings.ConfigFolders[0].Create("debug_log.txt")
	if err != nil {
		log.Panicf("Could not save logs, logging to console")
	}
	log.SetOutput(io.MultiWriter(os.Stdout, f))
	systray.Run(onReady, onExit)
}

func onReady() {
	InitLocale()
	settings.Load(&StoredSettings)

	systray.SetIcon(icon.Data)
	systray.SetTitle("Plex Rich Presence")
	systray.SetTooltip("Plex Rich Presence")
	timeMenu := systray.AddMenuItem(Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "TimeMenu",
			Other: "Time display",
		}}), Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "TimeMenuDescription",
			Other: "The way of displaying time in Discord",
		}}))
	timeElapsed := timeMenu.AddSubMenuItemCheckbox(Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "ElapsedMode",
			Other: "Elapsed",
		}}), Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "ElapsedModeDescription",
			Other: "Will show the elapsed time in Discord",
		}}), StoredSettings.TimeMode == "elapsed")
	timeRemaining := timeMenu.AddSubMenuItemCheckbox(Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "RemainingMode",
			Other: "Remaining",
		}}), Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "RemainingModeDescription",
			Other: "Will show the remaining in Discord",
		}}), StoredSettings.TimeMode == "remaining")
	autoLaunch := systray.AddMenuItemCheckbox(Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "AutoLaunch",
			Other: "Start on login",
		}}), Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "AutoLaunchDescription",
			Other: "Enable the automatic launch of the program when starting your computer",
		}}), appAutoStart.IsEnabled())
	systray.AddSeparator()
	disconnectBtn := systray.AddMenuItem(Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "Disconnect",
			Other: "Disconnect",
		}}), Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "DisconnectDescription",
			Other: "Disconnect from Plex. Will immediately trigger the opening of the browser to reconnect",
		}}))
	systray.AddSeparator()
	quitBtn := systray.AddMenuItem(Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "Quit",
			Other: "Quit",
		}}), Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "QuitDescription",
			Other: "Close the app and stop the rich presence from Plex",
		}}))
	ctx, cancelMain := context.WithCancel(context.Background())
	go mainFunc(ctx)
	for {
		select {
		case <-timeElapsed.ClickedCh:
			toggleTimeMode(timeElapsed, timeRemaining, "elapsed")
		case <-timeRemaining.ClickedCh:
			toggleTimeMode(timeRemaining, timeElapsed, "remaining")
		case <-autoLaunch.ClickedCh:
			if autoLaunch.Checked() {
				autoLaunch.Uncheck()
				appAutoStart.Disable()
			} else {
				autoLaunch.Check()
				appAutoStart.Enable()
			}
		case <-disconnectBtn.ClickedCh:
			StoredSettings.AccessToken = ""
			StoredSettings.Pin = PinSettings{}
			settings.Save(StoredSettings)
			cancelMain()
			go mainFunc(ctx)
		case <-quitBtn.ClickedCh:
			systray.Quit()
		}
	}
}

func toggleTimeMode(checkbox1 *systray.MenuItem, checkbox2 *systray.MenuItem, valToSet string) {
	checkbox1.Check()
	checkbox2.Uncheck()
	StoredSettings.TimeMode = valToSet
	settings.Save(StoredSettings)
}

func mainFunc(ctx context.Context) {

	var Plex *plex.Plex
	var filteredServers []plex.PMSDevices
	var accountData plex.UserPlexTV
	var runningSockets []*chan interface{}
	var wg sync.WaitGroup

	timeoutchan := make(chan bool)

	for {
		log.Printf("Refreshing servers")
		Plex = GetPlexTv()
		accountData, _ = Plex.MyAccount()
		servers, err := Plex.GetServers()
		if err != nil {
			return
		}

		filteredServers = nil
		for _, server := range servers {
			wg.Add(1)
			go GetGoodURI(server, &filteredServers, &wg)
		}

		wg.Wait()

		if len(runningSockets) > 0 {
			for _, socket := range runningSockets {
				*socket <- true
			}
		}

		for _, server := range filteredServers {
			StartWebsocketConnections(server, &accountData, &runningSockets)
		}
		log.Printf("Sucessfully connected to found WebSocket links")
		go func() {
			<-time.After(3 * time.Minute)
			timeoutchan <- true
		}()

		select {
		case <-ctx.Done():
			return
		case <-timeoutchan:
		}
	}
}

func onExit() {
	LogoutDiscordClient()
	os.Exit(0)
}