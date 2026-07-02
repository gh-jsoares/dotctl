package bootstrap

import "github.com/BurntSushi/toml"

func tomlUnmarshal(data []byte, v any) error {
	return toml.Unmarshal(data, v)
}
