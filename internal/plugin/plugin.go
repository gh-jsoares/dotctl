package plugin

import "github.com/BurntSushi/toml"

type Plugin struct {
	Name        string     `toml:"name"`
	Description string     `toml:"description"`
	Dir         string     `toml:"-"`
	Hooks       Hooks      `toml:"hooks"`
	Conditions  Conditions `toml:"conditions"`
	Ordering    Ordering   `toml:"ordering"`
	Options     Options    `toml:"options"`
}

type Hooks struct {
	Sync      string `toml:"sync"`
	Bootstrap string `toml:"bootstrap"`
	Doctor    string `toml:"doctor"`
}

type Conditions struct {
	PathsExist    []string `toml:"paths_exist"`
	BinariesExist []string `toml:"binaries_exist"`
	BinariesAbsent []string `toml:"binaries_absent"`
	Contexts      []string `toml:"contexts"`
	Check         string   `toml:"check"`
}

type Ordering struct {
	After    []string `toml:"after"`
	Before   []string `toml:"before"`
	Priority int      `toml:"priority"`
}

type Options struct {
	ContinueOnError bool              `toml:"continue_on_error"`
	Sudo            bool              `toml:"sudo"`
	Workdir         string            `toml:"workdir"`
	Timeout         int               `toml:"timeout"`
	Env             map[string]string `toml:"env"`
}

func Parse(path string) (*Plugin, error) {
	var p Plugin
	if _, err := toml.DecodeFile(path, &p); err != nil {
		return nil, err
	}
	if p.Ordering.Priority == 0 {
		p.Ordering.Priority = 50
	}
	if p.Options.Timeout == 0 {
		p.Options.Timeout = 120
	}
	return &p, nil
}
