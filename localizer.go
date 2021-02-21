package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"os"

	"golang.org/x/text/language"

	"github.com/cubiest/jibberjabber"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Localizer is the instance we use for localizing
var Localizer *i18n.Localizer

//go:embed locales/*
var localeFiles embed.FS

// InitLocale prepares the Localizer object
func InitLocale() {
	bundle := i18n.NewBundle(language.English)
	userLanguage, err := jibberjabber.DetectLanguage()
	if err != nil {
		log.Printf("Cannot set language automatically (%s), setting to english", err)
	} else {
		log.Printf("Detected language: %s", userLanguage)
	}
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	fs.WalkDir(localeFiles, "locales", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			log.Printf("Could not open locale file: %s", err)
		}
		if entry.IsDir() {
			return nil
		}
		fileContent, _ := localeFiles.ReadFile(path)
		bundle.ParseMessageFileBytes(fileContent, entry.Name())
		return nil
	})
	Localizer = i18n.NewLocalizer(bundle, userLanguage)
}
