package settings

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"

	"github.com/shibukawa/configdir"
	"gitlab.com/Arno500/plex-richpresence/types"
)

// Copied from https://medium.com/@matryer/golang-advent-calendar-day-eleven-persisting-go-objects-to-disk-7caf1ee3d11d

var lock sync.Mutex
var configFile = "config.json"
var configDirs configdir.ConfigDir = configdir.New("Arno & Co", "Plex-Richpresence")
var ConfigFolders []*configdir.Config = configDirs.QueryFolders(configdir.Global)

// StoredSettings contains the global settings of the app
var StoredSettings = types.PlexRPSettings{
	EnableNewDevicesByDefault: true,
	Devices:                   []types.Device{},
}

// Save our config
// Save - Saves a representation of v to the file at path.
func Save() error {
	lock.Lock()
	defer lock.Unlock()
	f, err := ConfigFolders[0].Create(configFile)
	if err != nil {
		return err
	}
	defer f.Close()
	r, err := Marshal(&StoredSettings)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	return err
}

// Marshal - Replacable Marshal function
var Marshal = func(v interface{}) (io.Reader, error) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// Load loads the file at path into v.
// Use os.IsNotExist() to see if the returned error is due
// to the file being missing.
func Load() error {
	lock.Lock()
	defer lock.Unlock()
	folder := configDirs.QueryFolderContainsFile(configFile)
	if folder == nil {
		return nil
	}
	f, err := folder.Open(configFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return Unmarshal(f, &StoredSettings)
}

// Unmarshal is a function that unmarshals the data from the
// reader into the specified value.
// By default, it uses the JSON unmarshaller.
var Unmarshal = func(r io.Reader, v *types.PlexRPSettings) error {
	return json.NewDecoder(r).Decode(v)
}
