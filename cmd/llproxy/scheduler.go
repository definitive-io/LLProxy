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
	"math"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Response int

const (
	Ready = iota
	RateLimit
	RequestTooLarge
)

type ScheduledRequest struct {
	Request               *http.Request
	ResponseChannel       chan Response
	RequiredTokenCapacity float64
}

type Scheduler struct {
	Config          ModelConfig
	Provider        string
	Name            string
	Requests        chan ScheduledRequest
	Mu              sync.Mutex
	LastReqTime     time.Time
	RequestCapacity float64
	TokenCapacity   float64
}

type SchedulerMap map[string]*Scheduler

func initSchedulers(provider string, config map[string]ModelConfig) SchedulerMap {
	var schedulers = make(SchedulerMap)

	for name, schedulerConfig := range config {
		schedulers[name] = &Scheduler{
			Config:          schedulerConfig,
			Provider:        provider,
			Name:            name,
			Requests:        make(chan ScheduledRequest, schedulerConfig.MaxQueueSize),
			LastReqTime:     time.Now(),
			RequestCapacity: schedulerConfig.ReqsPerMinute,
			TokenCapacity:   schedulerConfig.TokensPerMinute,
		}
		go schedulers[name].run()
	}

	return schedulers
}

func (scheduler *Scheduler) run() {

	// Don't allow startup if a config is too low for the scheduler to operate
	if scheduler.Config.ReqsPerMinute <= 1 {
		zap.S().Fatalw("Scheduler rpm too low (<=1) ", "provider", scheduler.Provider, "scheduler", scheduler.Name, "rpm", scheduler.Config.ReqsPerMinute)
	}
	if scheduler.Config.TokensPerMinute <= 1 {
		zap.S().Fatalw("Scheduler tpm too low (<=1)", "provider", scheduler.Provider, "scheduler", scheduler.Name, "tpm", scheduler.Config.TokensPerMinute)
	}

	// Defensive coding, this shouldn't ever happen, but if it does this guarantees we'll restart the pod rather
	// than running without one of our schedulers.
	defer func() {
		if r := recover(); r != nil {
			zap.S().Fatalw("Unexpected Scheduler Error", "provider", scheduler.Provider, "scheduler", scheduler.Name, "error", r)
		}
	}()

	// A scheduler's task is to rate limit incoming calls
	zap.S().Infow("Scheduler Start", "provider", scheduler.Provider, "scheduler", scheduler.Name, "rpm", scheduler.Config.ReqsPerMinute, "tpm", scheduler.Config.TokensPerMinute)

	for {
		// Wait for the next active request to come in
		var request *ScheduledRequest
		select {
		case req := <-scheduler.Requests:
			request = &req

		case <-time.After(time.Second * 2.0):
			// If there's no request after 2 seconds go ahead and update our capacity, then resume waiting
			scheduler.updateCapacity()
			continue
		}

		// Requests that are too large should have been filtered out before now, but this ensures we'll never wait forever
		if request.RequiredTokenCapacity > scheduler.Config.TokensPerMinute {
			zap.S().Debugw("Rejecting request", "url", request.Request.URL, "tokens", request.RequiredTokenCapacity, "reason", "RequestTooLarge")
			request.ResponseChannel <- RequestTooLarge
			continue
		}

		// We have a request, wait until we have sufficient capacity
		scheduler.waitForCapacity(request)

		// Allocate capacity to our request and prepare for our next request
		zap.S().Infow("Handling request", "url", request.Request.URL, "tokens", request.RequiredTokenCapacity)
		scheduler.TokenCapacity -= request.RequiredTokenCapacity
		scheduler.RequestCapacity -= 1

		// Send a signal back to the caller that the request can proceed
		request.ResponseChannel <- Ready
	}
}

func (scheduler *Scheduler) updateCapacity() {
	now := time.Now()
	if scheduler.TokenCapacity < scheduler.Config.TokensPerMinute || scheduler.RequestCapacity < scheduler.Config.ReqsPerMinute {
		elapsed := now.Sub(scheduler.LastReqTime).Minutes()
		tokenCapacity := scheduler.TokenCapacity + elapsed*float64(scheduler.Config.TokensPerMinute)
		requestCapacity := scheduler.RequestCapacity + elapsed*float64(scheduler.Config.ReqsPerMinute)

		scheduler.TokenCapacity = math.Min(tokenCapacity, scheduler.Config.TokensPerMinute)
		scheduler.RequestCapacity = math.Min(requestCapacity, scheduler.Config.ReqsPerMinute)

		zap.S().Debugw("Scheduler Capacity", "provider", scheduler.Provider, "scheduler", scheduler.Name, "tokens", scheduler.TokenCapacity, "requests", scheduler.RequestCapacity)
	}
	scheduler.LastReqTime = now

}

func (scheduler *Scheduler) waitForCapacity(request *ScheduledRequest) {
	const epsilon = 0.1
	for {

		// Check if we have capacity for the request
		scheduler.updateCapacity()

		// Time until we have a free request, sufficient tokens, both
		var requestTime = math.Max(0.0, (1-scheduler.RequestCapacity)/scheduler.Config.ReqsPerMinute)
		var tokensTime = math.Max(0.0, (request.RequiredTokenCapacity-scheduler.TokenCapacity)/scheduler.Config.TokensPerMinute)
		var capacityTime = math.Max(requestTime, tokensTime)
		if capacityTime <= 0.0 {
			// We have capacity now
			return
		}

		// Otherwise sleep for between epsilon and 2 seconds, depending on how much capacity we need
		// This keeps the capacity numbers close to actual capacity for our metrics
		var sleepTime = time.Duration(math.Min(2.0, capacityTime+epsilon) * float64(time.Second))
		time.Sleep(sleepTime)
	}
}
