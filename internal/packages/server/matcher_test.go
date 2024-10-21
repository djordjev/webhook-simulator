package server

import (
	"bytes"
	"encoding/json"
	"github.com/djordjev/webhook-simulator/internal/packages/mapping"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

var payload = `
	{
		"user": {
			"name": {
				"firstName": "Jon",
				"lastName": "Doe"
			},
			"order": 1
		}
	}
`

var payloadNotMatching1 = `
	{
		"user": {
			"name": {
				"firstName": "JonNotMatcher",
				"lastName": "Doe"
			},
			"order": 1
		}
	}
`

var payloadNoFields = "{}"

func TestMatch(t *testing.T) {
	request, _ := http.NewRequest(http.MethodPost, "/randomPath1", bytes.NewBufferString(payload))
	request.Header.Set("Content-Type", "application/json")

	requestNoHeader, _ := http.NewRequest(http.MethodPost, "/randomPath1", bytes.NewBufferString(payload))

	requestNotMatching1, _ := http.NewRequest(http.MethodPost, "/randomPath1", bytes.NewBufferString(payloadNotMatching1))
	requestNotMatching1.Header.Set("Content-Type", "application/json")

	requestNoFields, _ := http.NewRequest(http.MethodPost, "/randomPath1", bytes.NewBufferString(payloadNoFields))
	requestNoFields.Header.Set("Content-Type", "application/json")

	var body map[string]any
	_ = json.Unmarshal([]byte(payload), &body)

	var bodyNotMatching map[string]any
	_ = json.Unmarshal([]byte(payloadNotMatching1), &bodyNotMatching)

	var bodyNoFields map[string]any
	_ = json.Unmarshal([]byte(payloadNoFields), &bodyNoFields)

	var flowPost = mapping.Flow{
		Request: &mapping.RequestDefinition{
			Method:  http.MethodPost,
			Path:    "/randomPath1",
			Body:    body,
			Headers: map[string]string{"Content-Type": "application/json"},
		},
	}

	var flowGet = mapping.Flow{
		Request: &mapping.RequestDefinition{
			Method:  http.MethodGet,
			Path:    "/randomPath1",
			Body:    body,
			Headers: map[string]string{"Content-Type": "application/json"},
		},
	}

	var flowPath2 = mapping.Flow{
		Request: &mapping.RequestDefinition{
			Method:  http.MethodPost,
			Path:    "/randomPath2",
			Body:    body,
			Headers: map[string]string{"Content-Type": "application/json"},
		},
	}

	testCases := []struct {
		name    string
		body    map[string]any
		flow    mapping.Flow
		request *http.Request
		isMatch bool
	}{
		{
			name:    "matches the payload",
			request: request,
			body:    body,
			flow:    flowPost,
			isMatch: true,
		},
		{
			name:    "does not match the payload if method is not matching",
			request: request,
			body:    body,
			flow:    flowGet,
			isMatch: false,
		},
		{
			name:    "does not match the payload if path is not matching",
			request: request,
			body:    body,
			flow:    flowPath2,
			isMatch: false,
		},
		{
			name:    "does not match the payload if body is not matching",
			request: requestNotMatching1,
			body:    bodyNotMatching,
			flow:    flowPost,
			isMatch: false,
		},
		{
			name:    "does not match the payload if no data in body",
			request: requestNoFields,
			body:    bodyNoFields,
			flow:    flowPost,
			isMatch: false,
		},
		{
			name:    "does not match the payload if header is not matching",
			request: requestNoHeader,
			body:    body,
			flow:    flowPost,
			isMatch: false,
		},
	}

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			matcher := RequestMatcher{
				request: v.request,
				flow:    &v.flow,
				body:    v.body,
			}

			matcher.Match()

			require.Equal(t, matcher.IsMatch(), v.isMatch)
		})
	}
}
