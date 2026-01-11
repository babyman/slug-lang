package util

import (
	"os"
	"path/filepath"

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
	Store        *ConfigStore
}

type ConfigStore struct {
	Values map[string]interface{}
}

func NewConfigStore(rootPath, slugHome string) *ConfigStore {
	store := &ConfigStore{
		Values: make(map[string]interface{}),
	}

	// Paths in precedence order (last one wins in a simple merge,
	// but here we apply from least to most specific).
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
