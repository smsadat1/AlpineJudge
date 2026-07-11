package shared

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var (
	AvailableRunners   []string
	AvailableLanguages map[string]map[string]struct{}
)

type Config struct {
	Languages map[string]Language `yaml:"languages"`
	Runners   []string            `yaml:"runners"`
}

type Language struct {
	Versions []string `yaml:"versions"`
}

func LoadConfigs(pathToConfig string) error {
	data, err := os.ReadFile(pathToConfig)
	if err != nil {
		return fmt.Errorf("Failed to open config(%v): %v", pathToConfig, err)
	}

	// allocate memory for the outer global map if it hasn't been initialized yet
	if AvailableLanguages == nil {
		AvailableLanguages = make(map[string]map[string]struct{})
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("Failed to unmarshal config(%v): %w", pathToConfig, err)
	}

	AvailableRunners = cfg.Runners

	for lang, info := range cfg.Languages {
		AvailableLanguages[lang] = make(map[string]struct{})

		for _, version := range info.Versions {
			AvailableLanguages[lang][version] = struct{}{}
		}
	}

	return nil
}

func IsLanguageSupported(lang, version string) bool {
	versions, ok := AvailableLanguages[lang]
	if !ok {
		return false
	}

	_, ok = versions[version]
	return ok
}
