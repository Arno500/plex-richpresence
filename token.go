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

	"gitlab.com/Arno500/plex-scrobbler/settings"

	"github.com/google/uuid"
	"github.com/jrudio/go-plex-client"
)

//TODO: Fix bug where it tries twice to get a PIN

// AppName contains the name of the application sent to Plex
var AppName = "Plex Scrobbler by Arno & Co"

type pinSettings struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}
type tokenResponse struct {
	Token string `json:"authToken"`
}

type plexScrobblerSettings struct {
	ClientIdentifier string
	AccessToken      string
	Pin              pinSettings
}

var storedSettings plexScrobblerSettings
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

}

func getClientIdentifier() string {
	if storedSettings.ClientIdentifier != "" {
		return storedSettings.ClientIdentifier
	}
	log.Printf("The client identifer has never been set. Generating one.")
	storedSettings.ClientIdentifier = uuid.NewString()
	settings.Save(storedSettings)
	return storedSettings.ClientIdentifier
}

// CheckToken prepares and check the token
func CheckToken() error {
	if storedSettings.AccessToken == "" {
		settings.Load(&storedSettings)
	}
	if storedSettings.AccessToken == "" {
		log.Printf("We never had an access token, generating it")
		err := retrieveToken(true)
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
	queryString.Set("X-Plex-Token", storedSettings.AccessToken)
	req.URL.RawQuery = queryString.Encode()

	resp, err := httpClient.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("The access token we have is outdated, getting another one")
		storedSettings.AccessToken = ""
		settings.Save(storedSettings)
		err := retrieveToken(true)
		if err != nil {
			return err
		}
		return nil
	}

	return nil
}

func retrieveToken(initialCheck bool) error {
	var tokenInformations plex.PinResponse
	if storedSettings.Pin.ID == 0 {
		log.Printf("We never had a pin on Plex, creating one")
		err := retrievePin()
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest("GET", "https://plex.tv/api/v2/pins/"+strconv.Itoa(storedSettings.Pin.ID), nil)
	req.Header.Add("accept", "application/json")
	queryString := req.URL.Query()
	queryString.Set("code", storedSettings.Pin.Code)
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
		return fmt.Errorf("%+v", tokenInformations.Errors)
	}
	if tokenInformations.AuthToken == "" {
		if initialCheck == true {
			log.Printf("The pin we previously had is corrupted or expired, getting a new one")
			storedSettings.Pin.Code = ""
			storedSettings.Pin.ID = 0
		} else {
			log.Printf("Waiting for user authentication")
			time.Sleep(1000 * time.Millisecond)
		}
		err := retrieveToken(false)
		return err
	}
	storedSettings.AccessToken = tokenInformations.AuthToken
	settings.Save(storedSettings)
	return nil
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

	storedSettings.Pin.Code = pinInformations.Code
	storedSettings.Pin.ID = pinInformations.ID
	settings.Save(storedSettings)
	plexAuthURL, nil := url.Parse("https://app.plex.tv/auth")
	plexAuthQuery := plexAuthURL.Query()
	plexAuthQuery.Set("clientID", storedSettings.ClientIdentifier)
	plexAuthQuery.Set("code", storedSettings.Pin.Code)
	plexAuthQuery.Set("context[device][product]", AppName)
	plexAuthURL.Fragment = "?" + plexAuthQuery.Encode()
	openbrowser(plexAuthURL.String())

	return nil
}

func getToken() string {
	err := CheckToken()
	if err != nil {
		log.Fatalln(err)
	}
	return storedSettings.AccessToken
}
