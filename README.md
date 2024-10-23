# webhook-simulator

![Test Results](https://github.com/djordjev/webhook-simulator/actions/workflows/test.yml/badge.svg?branch=main)

# A simple tool that simulates webhooks & http responses

## Overview

_webhook-simulator_ is a simple http server which can mock http responses as optionally fire a
request to a different endpoint. In order to work it needs:
- port on which it will listen for requests (by default it's 4488)
- A path to a local folder where configuration files are stored

It will listen for file system changes in specified folder and update mappings and responses
immediately so it doesn't need a restart. All configurations are stored in JSON files with
`.whs` extension. All files that have extension different than `.whs` will be ignored.

### Example of configuration file

```JSON
{
  "request": {
    "method": "POST",
    "path": "/propagate",
    "body": {
      "user": {
        "firstName": "Jon"
      }
    }
  },
  "response": {
    "delay": 300,
    "includeRequest": false,
    "code": 200,
    "headers": {
      "Content-Type": "${{header.Content-Type}}"
    },
    "body": {
      "user": {
        "lastName": "${{body.user.lastName}}",
        "onceMoreFirstName": "${{body.user.firstName}}",
        "another": "field2"
      }
    }
  },
  "web_hook": {
    "method": "GET",
    "path": "www.google.com",
    "delay": 200,
    "includeRequest": true,
    "headers": {},
    "payload": {
      "response": "${{body.user.firstName}}"
    }
  }
}
```

## Matching requests

Once started http server will listen for all incoming requests. For each request first step is 
to find a mapping that matches it. It will go through all `.whs` files in mapping directory 
and look at their `request` part. 

### Example of request part of config

```json
"request": {
    "method": "POST", // requires that incoming request is POST
    "path": "/propagate", // requires that incoming request have URL path /propagate
    "headers": { "x-api-key": "xyz" }, // requires following headers
    "body": { // requires that request body contains following JSON structure
      "user": {
        "firstName": "Jon"
      }
    }
  }
```

Request **can** have other fields in body or other http headers that are not specified in configuration
As long as it have **at least** those specified in configuration the request will be matched.

## Mocking response

Once the request is paired with configuration the server will return a response to it. Response
matching response will be used from configuration file 

### Example of response part of config

```json
"response": {
    "delay": 300, // How long (in miliseconds) server will wait to return a response
    "includeRequest": false, // if set to true server will include request payload body into response
    "code": 200, // status code that will be returned to client
    "headers": { // headers that will be returned to client
      "Content-Type": "${{header.Content-Type}}"
    },
    "body": { // response body to return. if `includeRequest` is true this will be merged into request payload
      "user": {
        "lastName": "${{body.user.lastName}}",
        "onceMoreFirstName": "${{body.user.firstName}}",
        "another": "field2"
      }
    }
  }
```

In response it's possible to replace particular value with one from request payload. It will be
explained in `Templating` section.

## Mocking web hooks

Once response is returned to client server can optionally trigger a new http request to 
another endpoint. This is _optional_ so if not specified only response will be returned.

### Example of webhook part of config

```json
"web_hook": {
    "method": "GET", // HTTP verb that will be used for webhook request
    "path": "www.google.com", // Endpoint it will hit with request
    "delay": 200, // optional delay before sending a request
    "includeRequest": true, // same meaning as in response section
    "headers": {}, // headers that will be used
    "payload": { // request body that will be used. If `includeRequest` is set it will be merged into payload body
      "response": "${{body.user.firstName}}"
    }
  }
```

## Templating

It's possible to use parts of request body or headers to construct response (or webhook request).
If some value should be replaced with value from request you can use

`"some_value": "body.user.firstName"` to get value from request body
`"some_value": "header.api-key"` to get value from request headers

_Note: currently templating does not work with arrays_

## Docker

Server can be run within Docker container. If using docker componse it's recommended to 
utilize volume binding from local machine to docker container to a folder that is used for 
mapping. That way you can externally change configurations and utilize file system listening.

```yaml
services:
  simulator:
    image: djvukovic/wh-simulaor:latest
    container_name: wh-simulator
    ports:
      - '4488:4488'
    volumes:
      - '/path/on/host:/mapping'
```

