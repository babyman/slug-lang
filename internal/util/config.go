package util

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Configuration struct {
	Version      string
	RootPath     string
	SlugHome     string
	Argv         []string
	DebugJsonAST bool
	DebugTxtAST  bool
	DefaultLimit int
	MainModule   string // The entry point module name (e.g., "slug.server")
	Store        *ConfigStore
}

type ConfigStore struct {
	Values map[string]interface{}
}

func NewConfigStore(rootPath, slugHome string, mainModule string, argv []string) *ConfigStore {
	store := &ConfigStore{
		Values: make(map[string]interface{}),
	}

	// Layer 1: Config Files (Lowest Precedence)
	searchPaths := []string{}
	if slugHome != "" {
		searchPaths = append(searchPaths, filepath.Join(slugHome, "lib", "slug.toml"))
	}
	if rootPath != "" {
		searchPaths = append(searchPaths, filepath.Join(rootPath, "slug.toml"))
	}

	for _, path := range searchPaths {
		var data map[string]interface{}
		if _, err := os.Stat(path); err == nil {
			if _, err := toml.DecodeFile(path, &data); err == nil {
				mergeMaps(store.Values, data, "")
			}
		}
	}

	// Layer 2: Environment Variables (SLUG__ prefix)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "SLUG__") {
			pair := strings.SplitN(env, "=", 2)
			if len(pair) == 2 {
				// SLUG__slug__server__port -> slug.server.port
				key := strings.TrimPrefix(pair[0], "SLUG__")
				key = strings.ReplaceAll(key, "__", ".")
				store.Values[key] = pair[1]
			}
		}
	}

	// Layer 3: CLI Parameters (using consistent parser)
	options, _ := ParseArgs(argv)
	for key, value := range options {
		resolvedKey := key
		// CLI Sugar: expand module-local keys if no dot is present
		if !strings.Contains(key, ".") && mainModule != "" && mainModule != "<main>" {
			resolvedKey = mainModule + "." + key
		}
		if len(value) == 1 {
			store.Values[resolvedKey] = value[0]
		} else {
			store.Values[resolvedKey] = value
		}
	}

	return store
}

func mergeMaps(dest map[string]interface{}, src map[string]interface{}, prefix string) {
	for k, v := range src {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		if subMap, ok := v.(map[string]interface{}); ok {
			mergeMaps(dest, subMap, fullKey)
		} else {
			dest[fullKey] = v
		}
	}
}

func (cs *ConfigStore) Get(key string) (interface{}, bool) {
	val, ok := cs.Values[key]
	return val, ok
}
