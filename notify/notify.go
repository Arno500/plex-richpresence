package notify

import (
	_ "embed"
	"log"
	"os"
	"path/filepath"

	"github.com/gen2brain/beeep"
)

//go:embed discord.png
var notifIcon []byte

// SendNotification on desktop
func SendNotification(title string, text string) {
	tempIconPath := filepath.Join(os.TempDir(), "prp-notif-icon.png")
	if _, err := os.Stat(tempIconPath); err != nil {
		if os.IsNotExist(err) {
			err = os.WriteFile(tempIconPath, notifIcon, 0644)
			if err != nil {
				log.Panic(err)
			}
		}
	}
	err := beeep.Notify(title, text, tempIconPath)
	if err != nil {
		log.Panic(err)
	}
}
