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
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {

	// Define a string flag for the configuration file path with a default value
	configFilePath := flag.String("config", "config.json", "path to the configuration file")

	// Parse the flags
	flag.Parse()

	// Load the configuration
	config := LoadConfig(*configFilePath)

	// Setup Logging
	ConfigureLogging(config.Logging.Type, config.Logging.Level)

	// In order to keep our health and readiness probes running while the server is shutting down we setup
	// separate handlers for health and readiness from our main http server.

	// Setup the providers and base routes
	providers := initProviders(&config)
	for route, handler := range providers {
		zap.S().Infof("creating route for /%s/", route)
		http.HandleFunc("/"+route, handler)
		http.HandleFunc("/"+route+"/", handler)
	}

	// Create http servers
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Application.Port),
		Handler: http.DefaultServeMux,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// Unexpected server shutdown
			zap.S().Fatalf("Server closed unexpectedly: %v", err)
		}
	}()

	// Setup health endpoints
	HealthStartup(&config)

	// Channel for os signals
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	// Channel for server shutdown
	serverShutdown := make(chan struct{})

	// Listen for os signals
	signalReceived := false
	go func() {
		for sigName := range sig {
			if signalReceived {
				zap.S().Fatal("Second signal received. Exiting immediately.")
			} else {
				signalReceived = true
				zap.S().Infof("Received signal %v. Draining requests and shutting down.", sigName)

				// Mark the server as not ready
				HealthShutdown()

				// Create a context for shutdown with timeout
				// We give a fairly long timeout since requests can take a while to generate and we want to allow them time
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()

				go func() {
					if err := server.Shutdown(ctx); err != nil {
						zap.S().Errorf("Server shutdown: %v", err)
					} else {
						zap.S().Info("Shutdown complete.")
					}
					serverShutdown <- struct{}{}
				}()
			}
		}
	}()

	// Wait for server to shutdown
	<-serverShutdown
}
