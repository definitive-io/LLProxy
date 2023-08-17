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
package main_test

import (
	"github.com/definitive-io/llproxy/cmd/llproxy"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Use the require package from testify to stop the test and fail it immediately if the error is not nil.
	require := require.New(t)

	// Create a temporary directory for the config file.
	tempDir := t.TempDir()

	// Define a path to the temporary config file.
	configPath := filepath.Join(tempDir, "config.json")

	// Define a dummy config for testing.
	configContent := `{
        "routes": {
            "/route1": {
                "forward": "http://forward1.com",
                "provider": "provider1",
                "models": {
                    "model1": {
                        "maxQueueSize": 100,
                        "maxQueueWait": 1.5,
                        "rpm": 1000,
                        "tpm": 10000,
                        "cpm": 100000
                    }
                }
            }
        }
    }`

	// Write the config content to the file.
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(err, "Failed to write to temporary file")

	// Use the loadConfig function to read the config.
	config := main.LoadConfig(configPath)

	// Assert that the loaded config matches the expected config.
	route1 := config.Routes["/route1"]
	require.NotNil(route1, "Route1 should not be nil")

	require.Equal("http://forward1.com", route1.Forward)
	require.Equal("provider1", route1.Provider)

	model1 := route1.Models["model1"]
	require.NotNil(model1, "Model1 should not be nil")

	require.Equal(100, model1.MaxQueueSize)
	require.Equal(1.5, model1.MaxQueueWait)
	require.Equal(1000.0, model1.ReqsPerMinute)
	require.Equal(10000.0, model1.TokensPerMinute)
	require.Equal(100000.0, model1.CharsPerMinute)

	// Default parameters
	require.Equal(main.LogType("console"), config.Logging.Type)
	require.Equal(main.LogLevel("info"), config.Logging.Level)
	require.Equal(8080, config.Application.Port)

}
