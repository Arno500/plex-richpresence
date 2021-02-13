package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"golang.org/x/text/language"

	"github.com/cloudfoundry/jibber_jabber"
	"github.com/markbates/pkger"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Localizer is the instance we use for localizing
var Localizer *i18n.Localizer

// InitLocale prepares the Localizer object
func InitLocale() {
	bundle := i18n.NewBundle(language.English)
	userLanguage, _ := jibber_jabber.DetectLanguage()
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	pkger.Walk("/locales", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		fr, _ := pkger.Open(path)
		bytes, _ := ioutil.ReadAll(fr)
		bundle.ParseMessageFileBytes(bytes, info.Name())

		return nil
	})
	Localizer = i18n.NewLocalizer(bundle, userLanguage)
}
