package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	Port    int
	Workers int
	Debug   bool
}

type ServersConfig struct {
	Servers []string `json:"servers"`
}

func GetServersConfig(filepath string) (*ServersConfig, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config ServersConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
