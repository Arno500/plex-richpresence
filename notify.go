package main

import "github.com/gen2brain/beeep"

func SendNotification(title string, text string) {
	err := beeep.Notify(title, text, "assets/information.png")
	if err != nil {
		panic(err)
	}
}
