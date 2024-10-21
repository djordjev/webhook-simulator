package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/djordjev/webhook-simulator/internal/packages/mapping"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

var payloadReq = `
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

var payloadRes = `
	{
		"user": {
			"name": {
				"firstName": "Jon",
				"lastName": "Doe",
				"middle": "unknown"
			},
			"age": 35,
			"order": 1
		},
		"hello": {
			"nested": "world",
			"againFirstName": "Jon"
		}
	}
`

func TestResponder(t *testing.T) {
	request, _ := http.NewRequest(http.MethodPost, "/randomPath1", bytes.NewBufferString(payloadReq))
	request.Header.Set("Content-Type", "application/json")

	var body map[string]any
	_ = json.Unmarshal([]byte(payloadReq), &body)

	templateBody := map[string]any{
		"hello": map[any]any{"nested": "world", "againFirstName": "${{body.user.name.firstName}}"},
		"user":  map[any]any{"age": 35, "name": map[any]any{"middle": "unknown"}},
	}

	var flowPostNoWebhook = mapping.Flow{
		Response: &mapping.ResponseDefinition{
			Code:           http.StatusOK,
			IncludeRequest: true,
			Body:           templateBody,
			Headers:        map[string]string{"Content-Type": "application/json"},
		},
	}

	testCases := []struct {
		name                 string
		request              *http.Request
		response             *httptest.ResponseRecorder
		flow                 *mapping.Flow
		body                 map[string]any
		expectedStatusCode   int
		expectedResponseBody string
	}{
		{
			name:                 "responds to request but does not trigger webhook - with includeRequest",
			request:              request,
			response:             httptest.NewRecorder(),
			flow:                 &flowPostNoWebhook,
			body:                 body,
			expectedStatusCode:   http.StatusOK,
			expectedResponseBody: payloadRes,
		},
	}

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			responder := RequestResponder{
				request: v.request,
				flow:    v.flow,
				body:    v.body,
				rw:      v.response,
				mainCtx: ctx,
			}

			responder.Respond()

			require.Equal(t, v.response.Code, v.expectedStatusCode)
			require.JSONEq(t, payloadRes, v.response.Body.String())
		})
	}

}
