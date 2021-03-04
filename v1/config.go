package zplorama

import (
	_ "embed" // We bake the config file into the executable
	"os"

	"github.com/yosuke-furukawa/json5/encoding/json5"
)

//go:embed config.json
var configJSON []byte

// Config contains app-level configuration
var Config ConfStruct

// LoadConfig Loads an alternate config json file
func LoadConfig(filename string) error {
	contents, err := os.ReadFile(filename)

	if err != nil {
		return err
	}

	err = json5.Unmarshal(contents, &Config)

	return err
}

func init() {
	json5.Unmarshal(configJSON, &Config)
}
