# LLProxy - Large Language Proxy
![LLProxy](img/splash.png)


[![Release Notes](https://img.shields.io/github/release/definitive-io/llproxy)](https://github.com/definitive-io/llproxy/releases)
![GitHub Repo stars](https://img.shields.io/github/stars/definitive-io/LLProxy?style=social)
[![Open Issues](https://img.shields.io/github/issues-raw/definitive-io/llproxy)](https://github.com/definitive-io/llproxy/issues)
![GitHub Go version](https://img.shields.io/github/go-mod/go-version/definitive-io/llproxy?color=green)
[![License: Apache 2.0](https://img.shields.io/github/license/definitive-io/llproxy)](https://opensource.org/licenses/Apache-2-0)
[![Twitter](https://img.shields.io/twitter/url/https/twitter.com?style=social&label=Follow%20%40DefinitiveIO)](https://twitter.com/definitiveio)
[![Discord](https://dcbadge.vercel.app/api/server/CPJJfq87Vx?compact=true&style=flat)](https://discord.gg/CPJJfq87Vx)


## Summary
`LLProxy` was designed for the task of effectively managing rate limits and scheduling of workload across multiple different LLM based applications.  The rate limits for these services are complex, beyond what can easily be configured with the simplest of reverse proxies.  `LLProxy` addresses this by creating a scheduler that deeply understandings the core LLM providers rate limiting behavior.

## Features
* The following providers are currently supported: [`openai`]
* The following scheduling is currently supported: [`FIFO`]


## Usage

1. Setup your configuration file:

    ```bash
    cp config-example.json config.json
    ```

    Each provider can be defined as a specific route.
    
    `config.json`
    ```
    {
        "routes": {
            "openai": {
                "forward": "https://api.openai.com",
                "provider": "openai",
                "models": {
                    "gpt-4": {
                        "maxQueueSize": 10,
                        "maxQueueWait": 30,
                        "rpm": 200,
                        "tpm": 40000
                    },
                    ...
                }
            }
            ...
        }
    }
    ```
    The above creates a route http://proxyhost:8080/openai/... that routes all traffic sent to that route to https://api.openai.com/...

    It further defines a scheduler for the gpt-4 model that sets:
    * `maxQueueSize` defines how many requests are allowed to sit in the queue prior to being scheduled
    * `maxQueueWait` defines how long, in seconds, it will allow a request to wait before it starts rejecting additional requests with `RateLimit` errors.
    * `rpm` the maximum requests per minute
    * `tpm` the maximum tokens per minute

    Requests and tokens per minute are consumed as requests come in and recover over time.  If a request cannot be immediately processed then it will sit in the queue for up to `maxQueueWait` seconds, and up to `maxQueueSize` items can be outstanding in the queue.

    Set a config for every model you want to support.

1. [Optional] Run tests

    ```sh
    ./test.sh
    ```

1. [Optional] Look at code coverage
   
    ```
    go tool cover -html=coverage.out -o coverage.html
    ``` 

1. Build the application

    ```sh
    ./build.sh
    ```

1. Run the application

    ```sh
    ./llproxy
    ```

1. Direct traffic to your proxy server

    ```python
    import openai
    openai.api_base = 'http://<your-proxy-address>:8080/openai/v1'
    ...
    ```

----
