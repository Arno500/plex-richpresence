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
	"gopkg.in/natefinch/lumberjack.v2"

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
	log.SetOutput(io.MultiWriter(&lumberjack.Logger{
		Filename:   filepath.Join(settings.ConfigFolders[0].Path, "debug_log.txt"),
		MaxSize:    5,
		MaxBackups: 3,
		MaxAge:     5,
		Compress:   true,
	}, os.Stdout))
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
	defer func() {
		if r := recover(); r != nil {
			log.Println("System error, retrying in 10 seconds", r)
			time.Sleep(5 * time.Second)
			mainFunc(ctx)
		}
	}()
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

func disconnectSockets(sockets *[]*chan interface{}) {
	if len(*sockets) > 0 {
		for _, socket := range *sockets {
			select {
			case *socket <- true:
			default:
			}
		}
	}
	*sockets = nil
}

func mainFunc(ctx context.Context) {

	var Plex *plex.Plex
	var filteredServers []plex.PMSDevices
	var accountData plex.UserPlexTV
	var runningSockets []*chan interface{}

	timeoutchan := make(chan bool)

	// TODO: Need to seed first session data if admin (Plex doesn't resend data)

	for {
		var wg sync.WaitGroup
		log.Printf("Refreshing servers")
		Plex = GetPlexTv()
		accountData, _ = Plex.MyAccount()
		servers, _ := Plex.GetServers()

		filteredServers = nil
		for _, server := range servers {
			wg.Add(1)
			go GetGoodURI(server, &filteredServers, &wg)
		}

		wg.Wait()

		disconnectSockets(&runningSockets)

		for _, server := range filteredServers {
			go StartWebsocketConnections(server, &accountData, &runningSockets)
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
	LogoutDiscordClient()
	os.Exit(0)
}
