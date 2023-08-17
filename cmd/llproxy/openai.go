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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// Currently assumed "most recent" versions for token count assumptions
const GPT_3_5_DEFAULT = "gpt-3.5-turbo-0613"
const GPT_4_DEFAULT = "gpt-4-0613"

type OpenAIProvider struct {
	client     HttpClient
	urlBase    string
	schedulers SchedulerMap
}

// Wrap these so that we can define our Request interface
type AudioRequest openai.AudioRequest
type ChatCompletionRequest openai.ChatCompletionRequest
type CompletionRequest openai.CompletionRequest
type EmbeddingRequest openai.EmbeddingRequest
type EditsRequest openai.EditsRequest
type Request interface {
	TokensForRequest() (int, error)
}

func NewOpenAI(config *RouteConfig, client HttpClient) *OpenAIProvider {
	if config.Provider != "openai" {
		// Never expected to actually happen in normal operation
		zap.S().Fatalf("Initializing OpenAI provider with config for %s", config.Provider)
	}

	/*
		TODO: May make more sense to read limits from https://api.openai.com/dashboard/rate_limits
		Potential reason not to: this api is not documented and may change/go away
	*/
	return &OpenAIProvider{
		client:     client,
		schedulers: initSchedulers(config.Provider, config.Models),
		urlBase:    config.Forward,
	}
}

func (o *OpenAIProvider) GetHandler() func(http.ResponseWriter, *http.Request) {
	// Create the closure for the handler function with this Provider
	return func(w http.ResponseWriter, r *http.Request) {

		// Find the model for the request
		model, request, err := o.ParseRequest(r)
		if err != nil {
			zap.S().Debugw("Bad Request", "url", r.URL, "reason", err.Error())
			http.Error(w, fmt.Sprintf("LLProxy: %s", err.Error()), http.StatusBadRequest)
			return
		}

		// If we have a model, pass the request to the matching scheduler
		// otherwise we can skip the scheduler and forward directly
		if model != "" {

			// Find the corresponding scheduler
			scheduler, ok := o.schedulers[model]
			if !ok {
				zap.S().Debugw("Rejecting request", "url", r.URL, "model", model, "reason", "NoSchedulerForModel")
				http.Error(w, fmt.Sprintf("LLMProxy: No scheduler found for model '%s'", model), http.StatusBadRequest)
				return
			}

			tokens, err := request.TokensForRequest()
			if err != nil {
				zap.S().Debugw("Rejecting request", "url", r.URL, "model", model, "reason", "TokensForRequestError")
				http.Error(w, "LLMProxy: could not extract tokens for request", http.StatusBadRequest)
				return
			}

			// Ensure that the schedule is capable of handling a request of this size
			if scheduler.Config.ReqsPerMinute < 1 || scheduler.Config.TokensPerMinute < float64(tokens) {
				zap.S().Debugw("Rejecting request", "url", r.URL, "model", model, "tokens", tokens, "reason", "RequestTooLarge")
				http.Error(w, fmt.Sprintf("LLProxy: Request too large for model '%s'", model), http.StatusBadRequest)
				return
			}

			// Create a ScheduledRequest and send it to the scheduler
			responseChannel := make(chan Response)
			scheduler.Requests <- ScheduledRequest{
				Request:               r,
				ResponseChannel:       responseChannel,
				RequiredTokenCapacity: float64(tokens),
			}

			// Wait for the scheduler to signal that we can proceed
			response := <-responseChannel

			// If we got a RateLimit response send that back to the client
			if response == RateLimit {
				zap.S().Debugw("Rejecting request", "url", r.URL, "model", model, "tokens", tokens, "reason", "RateLimit")
				http.Error(w, fmt.Sprintf("LLMProxy: RateLimit exceeded for model '%s'", model), http.StatusTooManyRequests)
				return
			} else if response == RequestTooLarge {
				// We should detected this before we scheduled the request, this shouldn't occur with normal expectations.
				zap.S().Debugw("Rejecting request", "url", r.URL, "model", model, "tokens", tokens, "reason", "RequestTooLarge")
				http.Error(w, fmt.Sprintf("LLProxy: Request too large for model '%s'", model), http.StatusBadRequest)
			}
		}

		// Forward the request to the service
		err = forwardRequest(o.client, o.urlBase, w, r)
		if err != nil {
			// TODO: May be worth more details here like the request id and other identifiers from openai
			zap.S().Infow("Provider Error", "url", r.URL, "model", model, "reason", err.Error())
			http.Error(w, fmt.Sprintf("LLMProxy: Error forwarding request: %s", err.Error()), http.StatusServiceUnavailable)
			return
		}
	}
}

func (o *OpenAIProvider) ParseRequest(r *http.Request) (model string, request Request, err error) {

	// Openai rate limits by Model:
	// 1. Only POST methods have rate limits
	// 2. `model` is mostly a body parameter of the same name.
	// There are the following exceptions
	// *  /v1/images/*  - does not have a model parameter, implied model is `DALL-E 2`
	// *  /v1/files     - does not have a model, perhaps no rate limit?
	// *  /v1/fine-tunes - has a model parameter, but it's usage of that is different
	// *  /v1/moderations - has a model parameter, but there is no rate limit

	if r.Method != http.MethodPost {
		return
	}

	// Read the body out of the request, then add it back to the message so we can read it later since ioutil will exhaust the buffer
	bodyRaw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", nil, fmt.Errorf("error reading request body: %w", err)
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyRaw))

	// Parse the body depending on what endpoint we are hitting
	switch {
	case strings.Contains(r.URL.Path, "/v1/files"):
		return

	case strings.Contains(r.URL.Path, "/v1/fine-tunes"):
		return

	case strings.Contains(r.URL.Path, "/v1/moderations"):
		return

	case strings.Contains(r.URL.Path, "/v1/images"):
		// TODO: Could split this out into the three request types for parsing, but not currently import to us
		return "DALL-E 2", nil, nil

	case strings.Contains(r.URL.Path, "/v1/audio"):
		request := new(AudioRequest)
		err = json.Unmarshal(bodyRaw, request)
		if err != nil {
			return "", nil, fmt.Errorf("error reading request body, %s: %w", r.URL.Path, err)
		}
		return request.Model, request, nil

	case strings.HasSuffix(r.URL.Path, "/v1/chat/completions"):
		request := new(ChatCompletionRequest)
		err = json.Unmarshal(bodyRaw, request)
		if err != nil {
			return "", nil, fmt.Errorf("error reading request body, %s: %w", r.URL.Path, err)
		}
		return request.Model, request, nil

	case strings.HasSuffix(r.URL.Path, "/v1/completions"):
		request := new(CompletionRequest)
		err = json.Unmarshal(bodyRaw, request)
		if err != nil {
			return "", nil, fmt.Errorf("error reading request body, %s: %w", r.URL.Path, err)
		}
		return request.Model, request, nil

	case strings.HasSuffix(r.URL.Path, "/v1/embeddings"):
		request := new(EmbeddingRequest)
		err = json.Unmarshal(bodyRaw, request)
		if err != nil {
			return "", nil, fmt.Errorf("error reading request body, %s: %w", r.URL.Path, err)
		}
		return request.Model.String(), request, nil

	case strings.HasSuffix(r.URL.Path, "/v1/edits"):
		zap.S().Warnw("deprecated OpenAI endpoint", "url", r.URL.Path)
		request := new(EditsRequest)
		err = json.Unmarshal(bodyRaw, request)
		if err != nil {
			return "", nil, fmt.Errorf("error reading request body, %s: %w", r.URL.Path, err)
		}
		return *request.Model, request, nil

	default:
		zap.S().Warnw("unexpected OpenAI endpoint", "url", r.URL.Path)
		return
	}
}

/*
Token Counting is based on OpenAI Cookbooks:
- https://github.com/openai/openai-cookbook/blob/main/examples/How_to_count_tokens_with_tiktoken.ipynb
- https://github.com/openai/openai-cookbook/blob/main/examples/api_request_parallel_processor.py
*/
func (r *AudioRequest) TokensForRequest() (numTokens int, err error) {
	return 1000, nil
}

func (r *ChatCompletionRequest) TokensForRequest() (numTokens int, err error) {
	// ChatCompletion is more complicated logic

	model := r.Model
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		return numTokens, fmt.Errorf("encoding for model: %v", err)
	}

	// If the model version hasn't been pinned, set it based on current most recent models
	if model == "gpt-3.5-turbo" {
		model = GPT_3_5_DEFAULT
		zap.S().Debugf("gpt-3.5-turbo may update over time. Returning num tokens assuming %s.", model)
	} else if model == "gpt-4" {
		model = GPT_4_DEFAULT
		zap.S().Debugf("gpt-4 may update over time. Returning num tokens assuming %s.", model)
	}

	var tokensPerMessage, tokensPerName, tokensPerRequest int

	tokensPerRequest = 3 // every reply is primed with <|start|>assistant<|message|>

	switch {
	case model == "gpt-3.5-turbo-0301":
		tokensPerMessage = 4 // every message follows <|start|>{role/name}\n{content}<|end|>\n
		tokensPerName = -1   // if there's a name, the role is omitted

	case model == "gpt-3.5-turbo-0613",
		model == "gpt-3.5-turbo-16k-0613",
		model == "gpt-4-0314",
		model == "gpt-4-32k-0314",
		model == "gpt-4-0613",
		model == "gpt-4-32k-0613":
		tokensPerMessage = 3
		tokensPerName = 1

	case strings.Contains(model, "gpt-3.5-turbo") || strings.Contains(model, "gpt-4"):
		zap.S().Warnf("%s is an unexpected version, tokens based on historical assumptions", model)
		tokensPerMessage = 3
		tokensPerName = 1

	default:
		err = fmt.Errorf("Unexpected model for chat completions: %s", model)
		return numTokens, err
	}

	for _, message := range r.Messages {
		numTokens += tokensPerMessage
		numTokens += len(tkm.Encode(message.Content, nil, nil))
		numTokens += len(tkm.Encode(message.Role, nil, nil))
		numTokens += len(tkm.Encode(message.Name, nil, nil))
		if message.Name != "" {
			numTokens += tokensPerName
		}
	}
	numTokens += tokensPerRequest

	// Add in response tokens, this is n * max_tokens
	n := r.N
	maxTokens := r.MaxTokens
	if n < 1 {
		n = 1
	}
	if maxTokens < 1 {
		// When maxTokens is not set in the request estimate 15
		// Based on openai cookbook:
		// https://github.com/openai/openai-cookbook/blob/main/examples/api_request_parallel_processor.py
		maxTokens = 15
	}
	numTokens += n * maxTokens

	return numTokens, nil
}

func (r *CompletionRequest) TokensForRequest() (numTokens int, err error) {

	return 1000, nil
}

func (r *EmbeddingRequest) TokensForRequest() (numTokens int, err error) {

	return 1000, nil
}

func (r *EditsRequest) TokensForRequest() (numTokens int, err error) {

	return 1000, nil
}
