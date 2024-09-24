package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Execution struct {
	Name          string `yaml:"name"`
	Authorization string `yaml:"authorization"`
	Type          string `yaml:"type"`
	Cadence       int    `yaml:"cadence"`
	Target        string `yaml:"target"`
}

type ServerConfig struct {
	Endpoint  string      `yaml:"endpoint"`
	Origin    string      `yaml:"origin"`
	Key       string      `yaml:"key"`
	Execution []Execution `yaml:"execution"`
}

func ReadServerConfig(l string, sc *ServerConfig) error {
	data, err := os.ReadFile(l)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &sc)
}
