package plex

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"

	"gitlab.com/Arno500/plex-richpresence/i18n"
	"gitlab.com/Arno500/plex-richpresence/notify"
	"gitlab.com/Arno500/plex-richpresence/settings"

	"github.com/Arno500/go-plex-client"
	"github.com/google/uuid"
	i18npkg "github.com/nicksnyder/go-i18n/v2/i18n"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

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
		err = fmt.Errorf("can't open links automatically on this platform")
	}
	if err != nil {
		log.Println(err)
		log.Printf("Here is the link: %s", url)
	}
	notify.SendNotification(i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "SignInNotificationTitle",
			Other: "Plex Rich Presence login",
		}}), i18n.Localizer.MustLocalize(&i18npkg.LocalizeConfig{
		DefaultMessage: &i18npkg.Message{
			ID:    "SignInNotificationDescription",
			Other: "You can now login in the browser window that just opened!",
		}}))

}

func getClientIdentifier() string {
	if settings.StoredSettings.ClientIdentifier != "" {
		return settings.StoredSettings.ClientIdentifier
	}
	log.Printf("The client identifier has never been set. Generating one.")
	settings.StoredSettings.ClientIdentifier = uuid.NewString()
	settings.Save()
	return settings.StoredSettings.ClientIdentifier
}

var waitingForAuth = sync.Mutex{}

// CheckToken prepares and check the token
func CheckToken() error {
	// Never allow concurrent execution of this one, wait for the other to resolve first
	waitingForAuth.Lock()
	defer waitingForAuth.Unlock()

	if settings.StoredSettings.AccessToken == "" {
		log.Printf("We never had an access token, generating it")
		err := retrieveToken(settings.StoredSettings.Pin.Code != "")
		if err != nil {
			return err
		}
	}
	req, _ := http.NewRequest("GET", "https://plex.tv/api/v2/user", nil)
	req.Header.Add("accept", "application/json")
	queryString := req.URL.Query()
	queryString.Set("X-Plex-Product", appName)
	queryString.Set("X-Plex-Client-Identifier", getClientIdentifier())
	queryString.Set("X-Plex-Token", settings.StoredSettings.AccessToken)
	req.URL.RawQuery = queryString.Encode()

	resp, err := httpClient.Do(req)

	if err != nil {
		log.Printf("The token may be bad:")
		log.Println(err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		log.Printf("The access token we have is outdated, getting another one")
		settings.StoredSettings.AccessToken = ""
		settings.Save()
		err := retrieveToken(true)
		if err != nil {
			return err
		}
	}

	return nil
}

func retrieveToken(forceBrowser bool) error {
	var tokenInformations plex.PinResponse
	if settings.StoredSettings.Pin.ID == 0 {
		log.Printf("We never had a pin on Plex, creating one")
		err := retrievePin()
		if err != nil {
			return err
		}
	}
	req, _ := http.NewRequest("GET", "https://plex.tv/api/v2/pins/"+strconv.Itoa(settings.StoredSettings.Pin.ID), nil)
	req.Header.Add("accept", "application/json")
	queryString := req.URL.Query()
	queryString.Set("code", settings.StoredSettings.Pin.Code)
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
		if forceBrowser {
			askForConnection()
		} else {
			log.Printf("Waiting for user authentication")
			time.Sleep(2000 * time.Millisecond)
		}
		err := retrieveToken(false)
		return err
	}
	settings.StoredSettings.AccessToken = tokenInformations.AuthToken
	settings.Save()
	return nil
}

func askForConnection() {
	plexAuthURL, _ := url.Parse("https://app.plex.tv/auth")
	plexAuthQuery := plexAuthURL.Query()
	plexAuthQuery.Set("clientID", settings.StoredSettings.ClientIdentifier)
	plexAuthQuery.Set("code", settings.StoredSettings.Pin.Code)
	plexAuthQuery.Set("context[device][product]", appName)
	plexAuthURL.Fragment = "?" + plexAuthQuery.Encode()
	openbrowser(plexAuthURL.String())
}

func retrievePin() error {
	var pinInformations plex.PinResponse

	req, _ := http.NewRequest("POST", "https://plex.tv/api/v2/pins", nil)
	req.Header.Add("accept", "application/json")
	queryString := req.URL.Query()
	queryString.Set("strong", "true")
	queryString.Set("X-Plex-Product", appName)
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

	settings.StoredSettings.Pin.Code = pinInformations.Code
	settings.StoredSettings.Pin.ID = pinInformations.ID
	settings.Save()

	askForConnection()

	return nil
}
