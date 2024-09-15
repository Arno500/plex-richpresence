package gui

import (
	"log"

	"github.com/nekr0z/systray"
	"gitlab.com/Arno500/plex-richpresence/settings"
	"gitlab.com/Arno500/plex-richpresence/types"
)

func ToggleAutoStart(trayItem *systray.MenuItem) {
	if trayItem.Checked() {
		trayItem.Uncheck()
		appAutoStart.Disable()
	} else {
		trayItem.Check()
		appAutoStart.Enable()
	}
}

func ToggleAutoEnableDevices(trayItem *systray.MenuItem) {
	if trayItem.Checked() {
		trayItem.Uncheck()
		settings.StoredSettings.EnableNewDevicesByDefault = false
	} else {
		trayItem.Check()
		settings.StoredSettings.EnableNewDevicesByDefault = true
	}
	settings.Save()
}

func ToggleDevice(menuItem types.DeviceMenuItem) {
	if menuItem.MenuItem.Checked() {
		menuItem.MenuItem.Uncheck()
		menuItem.Device.Enabled = false
		log.Printf("Disabled device %s, (%s, %s)", menuItem.Device.Identifier, menuItem.Device.Title, menuItem.Device.Product)
	} else {
		menuItem.MenuItem.Check()
		menuItem.Device.Enabled = true
		log.Printf("Enabled device %s, (%s, %s)", menuItem.Device.Identifier, menuItem.Device.Title, menuItem.Device.Product)
	}
	settings.Save()
}
