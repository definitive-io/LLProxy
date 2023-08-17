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

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogType string
type LogLevel string

// configure logging or panic
func ConfigureLogging(logType LogType, logLevel LogLevel) {
	var cfg zap.Config
	var bytes []byte
	switch logType {
	case "json":
		bytes = []byte(`{
			"level": "debug",
			"encoding": "json",
			"outputPaths": ["stdout"],
			"errorOutputPaths": ["stderr"],
			"encoderConfig": {
				"messageKey": "message",
				"levelKey": "severity",
				"timeKey": "timestamp",
				"timeEncoder": "rfc3339",
				"levelEncoder": "capital"
			}
		}`)
	case "console":
		bytes = []byte(`{
			"level": "debug",
			"encoding": "console",
			"outputPaths": ["stdout"],
			"errorOutputPaths": ["stderr"],
			"encoderConfig": {
				"messageKey": "message",
				"levelKey": "severity",
				"timeKey": "timestamp",
				"timeEncoder": "rfc3339",
				"levelEncoder": "capital"
			}
		}`)
	default:
		panic(fmt.Errorf("unknown log_type %s", logType))
	}
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		panic(err)
	}
	level, err := zapcore.ParseLevel(string(logLevel))
	if err != nil {
		panic(err)
	}
	cfg.Level = zap.NewAtomicLevelAt(level)
	zap.ReplaceGlobals(zap.Must(cfg.Build()))
}
