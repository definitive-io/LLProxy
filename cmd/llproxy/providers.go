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
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

// Wrapper interface for http.Client to enable mocking and testing
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Handlers map[string]func(http.ResponseWriter, *http.Request)

type Provider interface {
	GetHandler() func(http.ResponseWriter, *http.Request)
}

func initProviders(config *Config) Handlers {
	// A provider is a single service such as OpenAI
	// A single provider may have multiple models/schedulers backing it
	// Determining how to identify which scheduler to use within a provider
	// is provider specific and needs to be coded for each provider specifically
	var handlers = make(Handlers)
	var client = &http.Client{}

	// Initialize the queue state for each scheduler
	for route, routeConfig := range config.Routes {
		zap.S().Infow("Initializing Provider", "provider", routeConfig.Provider)
		switch routeConfig.Provider {
		case "openai":
			openai := NewOpenAI(&routeConfig, client)
			handlers[route] = openai.GetHandler()
		default:
			zap.S().Fatalf("Unexpected Provider: '%s'\nCurrently supported providers: [openai]", routeConfig.Provider)
		}
	}

	return handlers
}

func forwardRequest(client HttpClient, URLBase string, w http.ResponseWriter, r *http.Request) error {
	// The main Proxy code, used by all Providers

	// Create a new URL from the raw r.URL to modify it
	url, err := url.Parse(r.URL.String())
	if err != nil {
		zap.S().Errorw("URL parse error", "url", r.URL, "reason", err)
		return err
	}

	// Split the path into segments and strip off the first segment
	segments := strings.Split(url.Path, "/")
	if len(segments) < 2 {
		zap.S().Errorw("URL parse error", "url", url, "reason", "expected provider path")
		return fmt.Errorf("Invalid URL: %s", url)
	}
	newPath := strings.Join(segments[2:], "/")

	// Modify the URL's scheme and host to the target URL's
	targetURL, err := url.Parse(URLBase)
	if err != nil {
		zap.S().Errorw("Base URL parse error", "url", URLBase, "reason", "Bad Provider Base URL")
		return err
	}
	url.Scheme = targetURL.Scheme
	url.Host = targetURL.Host
	url.Path = newPath

	// Create a new request using http
	request, err := http.NewRequest(r.Method, url.String(), r.Body)
	if err != nil {
		zap.S().Errorw("Unable to form new request", "url", url, "reason", err)
		return err
	}

	// Copy the headers from the original request
	copyHeader(request.Header, r.Header)

	// Send the request via a client
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the response back to the original writer
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)

	return err
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
