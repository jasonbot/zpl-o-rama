package zplorama

import (
	_ "embed" // We bake the config file into the executable

	"github.com/yosuke-furukawa/json5/encoding/json5"
)

//go:embed config.json
var configJSON []byte

// Config contains app-level configuration
var Config confStruct

func init() {
	json5.Unmarshal(configJSON, &Config)
}
