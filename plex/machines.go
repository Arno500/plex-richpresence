package plex

import (
	"log"

	"gitlab.com/Arno500/plex-richpresence/gui"
	"gitlab.com/Arno500/plex-richpresence/settings"
	"gitlab.com/Arno500/plex-richpresence/types"
)

func MachineIsEnabled(machine types.PlexPlayerKey) bool {
	for _, item := range settings.StoredSettings.Devices {
		if item.Identifier == machine.MachineIdentifier {
			return item.Enabled
		}
	}
	settings.StoredSettings.Devices = append(settings.StoredSettings.Devices, types.Device{
		Identifier: machine.MachineIdentifier,
		Enabled:    settings.StoredSettings.EnableNewDevicesByDefault,
		Product:    machine.Product,
		Title:      machine.Title})
	log.Printf("Added new device %s (%s, %s)", machine.MachineIdentifier, machine.Title, machine.Product)
	settings.Save()
	gui.SetupMachines()
	return true
}
