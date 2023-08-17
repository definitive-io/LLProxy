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
	"fmt"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

var (
	isReady = &atomicBool{val: true}
)

func HealthStartup(c *Config) {
	// We run our health endpoints on a different http server so that we can continue accepting requests
	// while we are in the process of shutting down
	livenessMux := http.NewServeMux()
	livenessMux.HandleFunc("/healthz", getHealthZ())
	livenessMux.HandleFunc("/readyz", getReadyZ())
	livenessServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", c.Application.HealthPort),
		Handler: livenessMux,
	}

	go func() {
		if err := livenessServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.S().Fatal("Liveness server failed: ", err)
		}
	}()
}

func HealthShutdown() {
	isReady.Set(false)
}

func getHealthZ() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

func getReadyZ() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if isReady.Get() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Not Ready"))
		}
	}
}

type atomicBool struct {
	sync.RWMutex
	val bool
}

func (ab *atomicBool) Get() bool {
	ab.RLock()
	defer ab.RUnlock()
	return ab.val
}

func (ab *atomicBool) Set(val bool) {
	ab.Lock()
	defer ab.Unlock()
	ab.val = val
}
