/*
Copyright 2023 Definitive Intelligence, Inc

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type ModelConfig struct {
	MaxQueueSize    int     `json:"maxQueueSize"`
	MaxQueueWait    float64 `json:"maxQueueWait"`
	ReqsPerMinute   float64 `json:"rpm"`
	TokensPerMinute float64 `json:"tpm"`
	CharsPerMinute  float64 `json:"cpm"`
}

type RouteConfig struct {
	Forward  string                 `json:"forward"`
	Provider string                 `json:"provider"`
	Models   map[string]ModelConfig `json:"models"`
}

type LoggingConfig struct {
	Level LogLevel `json:"level"`
	Type  LogType  `json:"type"`
}

type AppConfig struct {
	Port       int `json:"port"`
	HealthPort int `json:"healthPort"`
}

type Config struct {
	Application AppConfig              `json:"app"`
	Logging     LoggingConfig          `json:"logging"`
	Routes      map[string]RouteConfig `json:"routes"`
}

func LoadConfig(configFilePath string) Config {

	// Read the configuration file
	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		panic(fmt.Errorf("Failed to read config file: %v", err))
	}

	// Unmarshal the JSON data into the rateLimitMap
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		panic(fmt.Errorf("Failed to parse config file: %v", err))
	}

	// Set default values
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Type == "" {
		config.Logging.Type = "console"
	}
	if config.Application.Port == 0 {
		config.Application.Port = 8080
	}
	if config.Application.HealthPort == 0 {
		config.Application.HealthPort = 8081
	}

	return config
}
