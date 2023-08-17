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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

const FAKE_BASE_URL = "https://fake-testing-host.com"
const TEST_MODEL = "gpt-3.5-turbo"

type MockHttpClient struct{}

func (m *MockHttpClient) Do(req *http.Request) (response *http.Response, err error) {
	zap.S().Infof("MockHttpClient %s", req.URL.Path)

	switch {
	case strings.HasSuffix(req.URL.Path, "/v1/chat/completions"):
		response = &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString("dummy response")),
			Header:     make(http.Header),
		}
	case strings.HasSuffix(req.URL.Path, "/v1/embeddings"):
		response = &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString("dummy embedding")),
			Header:     make(http.Header),
		}

	default:
		response = &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(bytes.NewBufferString("not found")),
			Header:     make(http.Header),
		}
	}

	return response, nil
}

func CreateOpenAI() *OpenAIProvider {
	client := &MockHttpClient{}
	config := &RouteConfig{
		Forward:  FAKE_BASE_URL,
		Provider: "openai",
		Models: map[string]ModelConfig{
			TEST_MODEL: {
				MaxQueueSize:    10,
				MaxQueueWait:    1.0,
				ReqsPerMinute:   60.0,
				TokensPerMinute: 60000.0,
			},
		},
	}

	return NewOpenAI(config, client)
}

func TestNewOpenAI(t *testing.T) {
	openai := CreateOpenAI()

	assert.NotNil(t, openai)
	assert.NotNil(t, openai.client)
	assert.Equal(t, FAKE_BASE_URL, openai.urlBase)
	assert.Contains(t, openai.schedulers, TEST_MODEL)
}

func TestGetHandler_BadRoute(t *testing.T) {
	ConfigureLogging(LogType("console"), LogLevel("debug"))
	openai := CreateOpenAI()

	handler := openai.GetHandler()

	req := httptest.NewRequest("POST", "http://localhost:8080/openai/badroute", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	// Here you can check the status code and body of the response
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "not found", string(body))
}

func TestGetChatHandler_NoBody(t *testing.T) {
	ConfigureLogging(LogType("console"), LogLevel("debug"))
	openai := CreateOpenAI()

	handler := openai.GetHandler()

	req := httptest.NewRequest("POST", "http://localhost:8080/openai/v1/chat/completions", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	// Here you can check the status code and body of the response
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "LLProxy: error reading request body, /openai/v1/chat/completions: unexpected end of JSON input\n", string(body))
}

func TestGetChatHandler_Good(t *testing.T) {
	ConfigureLogging(LogType("console"), LogLevel("debug"))
	openai := CreateOpenAI()

	handler := openai.GetHandler()

	var bodyStr = []byte(fmt.Sprintf(`{"model": "%s", "messages": [{"role": "system", "content": "test"}]}`, TEST_MODEL))

	req := httptest.NewRequest("POST", "http://localhost:8080/openai/v1/chat/completions", bytes.NewBuffer(bodyStr))
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	// Check the status code and body of the response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "dummy response", string(body))
}

func TestGetEmbeddingHandler_NoBody(t *testing.T) {
	ConfigureLogging(LogType("console"), LogLevel("debug"))
	openai := CreateOpenAI()

	handler := openai.GetHandler()

	req := httptest.NewRequest("POST", "http://localhost:8080/openai/v1/embeddings", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	// Here you can check the status code and body of the response
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "LLProxy: error reading request body, /openai/v1/embeddings: unexpected end of JSON input\n", string(body))
}

func TestGetEmbeddingHandler_Good(t *testing.T) {
	ConfigureLogging(LogType("json"), LogLevel("debug"))
	openai := CreateOpenAI()

	handler := openai.GetHandler()

	var bodyStr = []byte(fmt.Sprintf(`{"model": "%s", "messages": [{"role": "system", "content": "test"}]}`, TEST_MODEL))

	req := httptest.NewRequest("POST", "http://localhost:8080/openai/v1/embeddings", bytes.NewBuffer(bodyStr))
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	// Check the status code and body of the response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "dummy embedding", string(body))
}

func TestChatCompletionRequestTokensForRequest(t *testing.T) {

	request := &ChatCompletionRequest{
		Model: TEST_MODEL,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "test",
			},
		},
		MaxTokens: 1,
	}
	tokens, err := request.TokensForRequest()
	assert.NoError(t, err)
	assert.Equal(t, 9, tokens) // 1 token in message, 1 for the role, 1 token in response, 6 tokens of overhead

	request = &ChatCompletionRequest{
		Model: TEST_MODEL,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: "Who won the world series in 2020?",
			},
		},
		MaxTokens: 60,
		N:         1,
	}

	tokens, err = request.TokensForRequest()
	assert.NoError(t, err)
	assert.Equal(t, 87, tokens) // 18 tokens in message, 60 tokens in response, 9 tokens of overhead

}
