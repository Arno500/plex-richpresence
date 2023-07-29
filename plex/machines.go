package plex

import (
	"log"

	"gitlab.com/Arno500/plex-richpresence/gui"
	"gitlab.com/Arno500/plex-richpresence/settings"
	"gitlab.com/Arno500/plex-richpresence/types"

	Plex "github.com/Arno500/go-plex-client"
)

var devicesCache = make(map[string]Plex.PMSDevices)

func RefreshDevicesCache(plex *Plex.Plex) {
	devices, _ := plex.GetDevices()
	devicesCache = make(map[string]Plex.PMSDevices)
	for _, item := range devices {
		devicesCache[item.ClientIdentifier] = item
	}
}

func GetDevice(plex *Plex.Plex, clientIdentifier string) Plex.PMSDevices {
	device, exist := devicesCache[clientIdentifier]
	if !exist {
		RefreshDevicesCache(plex)
		device = devicesCache[clientIdentifier]
	}
	return device
}


func MachineIsEnabled(machine types.PlexPlayerKey) bool {
	if machine.Product == "" && machine.Title == "" {
		return false
	}
	for _, item := range settings.StoredSettings.Devices {
		if item.Identifier == machine.ClientIdentifier {
			return item.Enabled
		}
	}
	settings.StoredSettings.Devices = append(settings.StoredSettings.Devices, types.Device{
		Identifier: machine.ClientIdentifier,
		Enabled:    settings.StoredSettings.EnableNewDevicesByDefault,
		Product:    machine.Product,
		Title:      machine.Title})
	log.Printf("Added new device %s (%s, %s)", machine.ClientIdentifier, machine.Title, machine.Product)
	settings.Save()
	gui.SetupMachines()
	return settings.StoredSettings.EnableNewDevicesByDefault
}
