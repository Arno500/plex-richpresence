package main

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/gen2brain/beeep"
	"github.com/markbates/pkger"
)

// SendNotification on desktop
func SendNotification(title string, text string) {
	f, _ := pkger.Open("/icon/discord.png")
	defer f.Close()
	tempIconPath := filepath.Join(os.TempDir(), "ps-notif-icon.png")
	tempIcon, _ := os.Create(tempIconPath)
	defer tempIcon.Close()
	_, err := io.Copy(tempIcon, f)
	if err != nil {
		log.Panic(err)
	}
	err = beeep.Notify(title, text, tempIconPath)
	if err != nil {
		log.Panic(err)
	}
}
