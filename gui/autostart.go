package gui

import (
	"os"
	"path/filepath"

	"github.com/emersion/go-autostart"
)

var appExec, _ = os.Executable()
var appExecResolved, _ = filepath.EvalSymlinks(appExec)

var appAutoStart = &autostart.App{
	Name:        "Plex Rich Presence",
	DisplayName: "Plex Rich Presence",
	Exec:        []string{appExecResolved},
}
