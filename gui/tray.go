package gui

import (
	"context"
	"fmt"
	"reflect"

	"github.com/nekr0z/systray"
	i18npkg "github.com/nicksnyder/go-i18n/v2/i18n"

	"gitlab.com/Arno500/plex-richpresence/autoupdate"
	"gitlab.com/Arno500/plex-richpresence/i18n"
	"gitlab.com/Arno500/plex-richpresence/icon"
	"gitlab.com/Arno500/plex-richpresence/settings"
	"gitlab.com/Arno500/plex-richpresence/types"
)

var TrayHandlers = types.TrayHandlersStruct{}

func StartTray(onReady func(), onExit func()) {
	systray.Run(onReady, onExit)
}

func SetupTray() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("Plex Rich Presence")
	systray.SetTooltip("Plex Rich Presence")
	timeMenu := systray.AddMenuItem(i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "TimeMenu",
			Other: "Time display",
		}}), i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "TimeMenuDescription",
			Other: "The way of displaying time in Discord",
		}}))
	TrayHandlers.TimeElapsed = timeMenu.AddSubMenuItemCheckbox(i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "ElapsedMode",
			Other: "Elapsed",
		}}), i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "ElapsedModeDescription",
			Other: "Will show the elapsed time in Discord",
		}}), settings.StoredSettings.TimeMode == "elapsed")
	TrayHandlers.TimeRemaining = timeMenu.AddSubMenuItemCheckbox(i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "RemainingMode",
			Other: "Remaining",
		}}), i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "RemainingModeDescription",
			Other: "Will show the remaining in Discord",
		}}), settings.StoredSettings.TimeMode == "remaining")
	systray.AddSeparator()
	TrayHandlers.EnabledDeviceByDefault = systray.AddMenuItemCheckbox(i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "EnableDevicesByDefault",
			Other: "Enable new devices automatically",
		}}), i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "EnableDevicesByDefaultDescription",
			Other: "Should new discovered devices be enabled by default",
		}}), settings.StoredSettings.EnableNewDevicesByDefault)
	TrayHandlers.Devices = systray.AddMenuItem(i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "DevicesMenu",
			Other: "Devices",
		}}), i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "DevicesMenuDescription",
			Other: "Which device should be tracked",
		}}))
	systray.AddSeparator()
	TrayHandlers.AutoLaunch = systray.AddMenuItemCheckbox(i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "AutoLaunch",
			Other: "Start on login",
		}}), i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "AutoLaunchDescription",
			Other: "Enable the automatic launch of the program when starting your computer",
		}}), appAutoStart.IsEnabled())
	systray.AddSeparator()
	TrayHandlers.DisconnectBtn = systray.AddMenuItem(i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "Disconnect",
			Other: "Disconnect",
		}}), i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "DisconnectDescription",
			Other: "Disconnect from Plex. Will immediately trigger the opening of the browser to reconnect",
		}}))
	systray.AddSeparator()
	var versionItem = systray.AddMenuItemCheckbox(i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		TemplateData: map[string]string{"Version": autoupdate.Version},
		DefaultMessage: &i18npkg.Message{
			ID:    "Version",
			Other: "Version: {{.Version}}",
		}}), "", false)
	versionItem.Disable()
	TrayHandlers.QuitBtn = systray.AddMenuItem(i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "Quit",
			Other: "Quit",
		}}), i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "QuitDescription",
			Other: "Close the app and stop the rich presence from Plex",
		}}))
	SetupMachines()
}

var devicesMenuContext context.Context
var devicesMenuCancel context.CancelFunc
var menuItemsWithDevices []types.DeviceMenuItem

func SetupMachines() {
	if (devicesMenuCancel) != nil {
		devicesMenuCancel()
		for _, item := range menuItemsWithDevices {
			item.MenuItem.Hide()
		}
		menuItemsWithDevices = []types.DeviceMenuItem{}
	}
	devicesMenuContext, devicesMenuCancel = context.WithCancel(context.Background())
	cases := []reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(devicesMenuContext.Done()),
		},
	}
	for i, device := range settings.StoredSettings.Devices {
		handler := TrayHandlers.Devices.AddSubMenuItemCheckbox(fmt.Sprintf("%s (%s)", device.Title, device.Product), i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
			TemplateData: map[string]string{"Identifier": device.Title, "Product": device.Product},
			DefaultMessage: &i18npkg.Message{
				ID:    "DeviceDescription",
				Other: "Enable {{.Identifier}} running {{.Product}}",
			}}), device.Enabled)
		menuItemsWithDevices = append(menuItemsWithDevices, types.DeviceMenuItem{MenuItem: handler, Device: &settings.StoredSettings.Devices[i]})
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(handler.ClickedCh),
		})
	}
	go func() {
		for {
			index, _, _ := reflect.Select(cases)
			// First case, if the context is done, we can stop listening for the events and close the routine
			if index == 0 {
				return
			} else
			// Third case, if the user clicked on a device checkbox, we need to update the settings
			{
				checkBoxIndex := index - 1 // -1 because we have 1 cases before the checkboxes
				ToggleDevice(menuItemsWithDevices[checkBoxIndex])
			}
		}
	}()
}

func Quit() {
	systray.Quit()
}
