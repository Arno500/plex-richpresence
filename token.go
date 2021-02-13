package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"gitlab.com/Arno500/plex-richpresence/settings"

	"github.com/Arno500/go-plex-client"
	"github.com/google/uuid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var httpClient = &http.Client{}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("Can't open links automatically on this platform")
	}
	if err != nil {
		log.Println(err)
		log.Printf("Here is the link: %s", url)
	}
	SendNotification(Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "SignInNotificationTitle",
			Other: "Plex Rich Presence login",
		}}), Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "SignInNotificationDescription",
			Other: "You can now login in the browser window that just opened!",
		}}))

}

func getClientIdentifier() string {
	if StoredSettings.ClientIdentifier != "" {
		return StoredSettings.ClientIdentifier
	}
	log.Printf("The client identifier has never been set. Generating one.")
	StoredSettings.ClientIdentifier = uuid.NewString()
	settings.Save(StoredSettings)
	return StoredSettings.ClientIdentifier
}

// CheckToken prepares and check the token
func CheckToken() error {
	if StoredSettings.AccessToken == "" {
		log.Printf("We never had an access token, generating it")
		var err error
		if StoredSettings.Pin.Code != "" {
			err = retrieveToken(true)
		} else {
			err = retrieveToken(false)
		}
		if err != nil {
			return err
		}
		return nil
	}
	req, err := http.NewRequest("GET", "https://plex.tv/api/v2/user", nil)
	req.Header.Add("accept", "application/json")
	queryString := req.URL.Query()
	queryString.Set("X-Plex-Product", AppName)
	queryString.Set("X-Plex-Client-Identifier", getClientIdentifier())
	queryString.Set("X-Plex-Token", StoredSettings.AccessToken)
	req.URL.RawQuery = queryString.Encode()

	resp, err := httpClient.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("The access token we have is outdated, getting another one")
		StoredSettings.AccessToken = ""
		settings.Save(StoredSettings)
		err := retrieveToken(true)
		if err != nil {
			return err
		}
		return nil
	}

	return nil
}

func retrieveToken(forceBrowser bool) error {
	var tokenInformations plex.PinResponse
	if StoredSettings.Pin.ID == 0 {
		log.Printf("We never had a pin on Plex, creating one")
		err := retrievePin()
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest("GET", "https://plex.tv/api/v2/pins/"+strconv.Itoa(StoredSettings.Pin.ID), nil)
	req.Header.Add("accept", "application/json")
	queryString := req.URL.Query()
	queryString.Set("code", StoredSettings.Pin.Code)
	queryString.Set("X-Plex-Client-Identifier", getClientIdentifier())
	req.URL.RawQuery = queryString.Encode()

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&tokenInformations); err != nil {
		return err
	}
	if len(tokenInformations.Errors) > 0 {
		if tokenInformations.Errors[0].Code == 1020 {
			log.Printf("The pin we previously had is corrupted or expired, getting a new one")
			retrievePin()
		} else {
			return fmt.Errorf("%+v", tokenInformations.Errors)
		}
	}
	if tokenInformations.AuthToken == "" {
		if forceBrowser == true {
			askForConnection()
		} else {
			log.Printf("Waiting for user authentication")
			time.Sleep(2000 * time.Millisecond)
		}
		err := retrieveToken(false)
		return err
	}
	StoredSettings.AccessToken = tokenInformations.AuthToken
	settings.Save(StoredSettings)
	return nil
}

func askForConnection() {
	plexAuthURL, _ := url.Parse("https://app.plex.tv/auth")
	plexAuthQuery := plexAuthURL.Query()
	plexAuthQuery.Set("clientID", StoredSettings.ClientIdentifier)
	plexAuthQuery.Set("code", StoredSettings.Pin.Code)
	plexAuthQuery.Set("context[device][product]", AppName)
	plexAuthURL.Fragment = "?" + plexAuthQuery.Encode()
	openbrowser(plexAuthURL.String())
}

func retrievePin() error {
	var pinInformations plex.PinResponse

	req, err := http.NewRequest("POST", "https://plex.tv/api/v2/pins", nil)
	req.Header.Add("accept", "application/json")
	queryString := req.URL.Query()
	queryString.Set("strong", "true")
	queryString.Set("X-Plex-Product", AppName)
	queryString.Set("X-Plex-Client-Identifier", getClientIdentifier())
	req.URL.RawQuery = queryString.Encode()

	resp, err := httpClient.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&pinInformations); err != nil {
		return err
	}

	if len(pinInformations.Errors) > 0 {
		return fmt.Errorf("%+v", pinInformations.Errors)
	}

	StoredSettings.Pin.Code = pinInformations.Code
	StoredSettings.Pin.ID = pinInformations.ID
	settings.Save(StoredSettings)

	askForConnection()

	return nil
}
