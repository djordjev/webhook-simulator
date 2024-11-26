package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/djordjev/webhook-simulator/internal/packages/mapping"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

var payloadResponderReq = `
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

var payloadResponderReqWithArray = `
	{	
		"user": {
			"name": {
				"firstName": "Jon",
				"lastName": "Doe"
			}
		},
		"info": [{ "random": "thing" }]
	}
`

var payloadResponderReqArrayMapping = `
	{	
		"users": [
			{ "name": "JonA" },
			{ "name": "JonB" },
			{ "name": "JonC" }
		]
	}
`

var payloadResponderRes = `
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

var payloadResponderResWithArray = `
	{
		"user": {
			"name": {
				"firstName": "Jon",
				"lastName": "Doe",
				"middle": "unknown"
			},
			"age": 35
		},
		"hello": {
			"nested": "world",
			"againFirstName": "Jon"
		},
		"info": [
			{ 
				"random": "thing",
				"user": {
					"firstName": "Jon",
					"lastName": "Doe"
				}
			},
			"Jon Hardcoded",
			"Doe",
			42,
			true
		]
	}
`

var payloadResponderResArrayMapping = `
	{
		"mappedUsers": [
			{ "mappedName": "JonA", "additional": 5},
			{ "mappedName": "JonB", "additional": 5},
			{ "mappedName": "JonC", "additional": 5}
		]
	}
`

var payloadResponderNotIncludedBody = `
	{
		"user": {
			"name": {
				"middle": "unknown"
			},
			"age": 35
		},
		"hello": {
			"nested": "world",
			"againFirstName": "Jon"
		}
	}
`

type mockHttpClient struct {
	mock.Mock
}

func (m *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)

	return args.Get(0).(*http.Response), args.Error(1)
}

func TestResponder(t *testing.T) {
	request, _ := http.NewRequest(http.MethodPost, "/randomPath1", bytes.NewBufferString(payloadResponderReq))
	request.Header.Set("Content-Type", "application/json")

	requestWithArray, _ := http.NewRequest(http.MethodPost, "/randomPath1", bytes.NewBufferString(payloadResponderReqWithArray))
	requestWithArray.Header.Set("Content-Type", "application/json")

	requestMappedArray, _ := http.NewRequest(http.MethodPost, "/randomPath1", bytes.NewBufferString(payloadResponderReqArrayMapping))
	requestMappedArray.Header.Set("Content-Type", "application/json")

	var body map[string]any
	_ = json.Unmarshal([]byte(payloadResponderReq), &body)

	var bodyWithArray map[string]any
	_ = json.Unmarshal([]byte(payloadResponderReqWithArray), &bodyWithArray)

	var bodyMappedArray map[string]any
	_ = json.Unmarshal([]byte(payloadResponderReqArrayMapping), &bodyMappedArray)

	templateBody := map[string]any{
		"hello": map[any]any{"nested": "world", "againFirstName": "${{body.user.name.firstName}}"},
		"user":  map[any]any{"age": 35, "name": map[any]any{"middle": "unknown"}},
	}

	templateBodyArray := maps.Clone(templateBody)
	templateBodyArray["info"] = []any{
		map[string]any{
			"user": map[string]any{
				"firstName": "${{body.user.name.firstName}}",
				"lastName":  "${{body.user.name.lastName}}",
			},
		},
		"Jon Hardcoded",
		"${{body.user.name.lastName}}",
		42,
		true,
	}

	templateBodyMappedArray := map[string]any{
		"mappedUsers": map[any]any{
			"$each": map[any]any{
				"$field": "${{body.users}}",
				"$to": map[any]any{
					"mappedName": "${{iterator.name}}",
					"additional": 5,
				},
			},
		},
	}

	var flowPostNoWebhook = mapping.Flow{
		Response: &mapping.ResponseDefinition{
			Code:           http.StatusOK,
			IncludeRequest: true,
			Body:           templateBody,
			Headers:        map[string]string{"Content-Type": "application/json"},
		},
	}

	var flowPostMappedArray = mapping.Flow{
		Response: &mapping.ResponseDefinition{
			Code:           http.StatusOK,
			IncludeRequest: false,
			Body:           templateBodyMappedArray,
			Headers:        map[string]string{"Content-Type": "application/json"},
		},
	}

	var flowPostWithArrayNoWebhook = mapping.Flow{
		Response: &mapping.ResponseDefinition{
			Code:           http.StatusOK,
			IncludeRequest: true,
			Body:           templateBodyArray,
			Headers:        map[string]string{"Content-Type": "application/json"},
		},
	}

	var flowPostNoWebhookNoInclude = mapping.Flow{
		Response: &mapping.ResponseDefinition{
			Code:           http.StatusOK,
			IncludeRequest: false,
			Body:           templateBody,
			Headers:        map[string]string{"Content-Type": "application/json"},
		},
	}

	var flowPostWithWebhook = mapping.Flow{
		Response: &mapping.ResponseDefinition{
			Code:           http.StatusOK,
			IncludeRequest: false,
			Body:           map[string]any{"ok": "ok"},
			Headers:        map[string]string{"Content-Type": "application/json"},
		},
		WebHook: &mapping.WebHookDefinition{
			Method:         http.MethodPut,
			Path:           "/randomPutMethod/put",
			IncludeRequest: true,
			Headers:        map[string]string{"x-api-key": "abc"},
			Body:           templateBody,
		},
	}

	var flowPostWithWebhookNoInclude = mapping.Flow{
		Response: &mapping.ResponseDefinition{
			Code:    http.StatusOK,
			Body:    map[string]any{"ok": "ok"},
			Headers: map[string]string{"Content-Type": "application/json"},
		},
		WebHook: &mapping.WebHookDefinition{
			Method:  http.MethodPut,
			Path:    "/randomPutMethod/put",
			Headers: map[string]string{"x-api-key": "abc"},
			Body:    templateBody,
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
		shouldTriggerWebHook bool
		webhookRequestBody   string
	}{
		{
			name:                 "responds to request but does not trigger webhook - with includeRequest",
			request:              request,
			response:             httptest.NewRecorder(),
			flow:                 &flowPostNoWebhook,
			body:                 body,
			expectedStatusCode:   http.StatusOK,
			expectedResponseBody: payloadResponderRes,
		},
		{
			name:                 "responds to request with slice but does not trigger webhook - with includeRequest",
			request:              requestWithArray,
			response:             httptest.NewRecorder(),
			flow:                 &flowPostWithArrayNoWebhook,
			body:                 bodyWithArray,
			expectedStatusCode:   http.StatusOK,
			expectedResponseBody: payloadResponderResWithArray,
		},
		{
			name:                 "responds to request but does not trigger webhook - no includeRequest",
			request:              request,
			response:             httptest.NewRecorder(),
			flow:                 &flowPostNoWebhookNoInclude,
			body:                 body,
			expectedStatusCode:   http.StatusOK,
			expectedResponseBody: payloadResponderNotIncludedBody,
		},
		{
			name:                 "responds to request and triggers a webhook - with includeRequest",
			request:              request,
			response:             httptest.NewRecorder(),
			flow:                 &flowPostWithWebhook,
			body:                 body,
			expectedStatusCode:   http.StatusOK,
			expectedResponseBody: `{"ok": "ok"}`,
			shouldTriggerWebHook: true,
			webhookRequestBody:   payloadResponderRes,
		},
		{
			name:                 "responds to request and triggers a webhook - no includeRequest",
			request:              request,
			response:             httptest.NewRecorder(),
			flow:                 &flowPostWithWebhookNoInclude,
			body:                 body,
			expectedStatusCode:   http.StatusOK,
			expectedResponseBody: `{"ok": "ok"}`,
			shouldTriggerWebHook: true,
			webhookRequestBody:   payloadResponderNotIncludedBody,
		},
		{
			name:                 "responds with mapped array",
			request:              requestMappedArray,
			response:             httptest.NewRecorder(),
			flow:                 &flowPostMappedArray,
			body:                 bodyMappedArray,
			expectedStatusCode:   http.StatusOK,
			expectedResponseBody: payloadResponderResArrayMapping,
		},
	}

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mocked := mockHttpClient{}

			responder := RequestResponseBuilder(
				v.request,
				v.flow,
				v.body,
				v.response,
				ctx,
				&mocked,
			)

			if v.shouldTriggerWebHook {
				response := http.Response{Body: io.NopCloser(bytes.NewBufferString("OK"))}
				mocked.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					pl := make(map[string]any)
					expected := make(map[string]any)

					_ = json.NewDecoder(req.Body).Decode(&pl)

					_ = json.Unmarshal([]byte(v.webhookRequestBody), &expected)

					return reflect.DeepEqual(expected, pl)
				})).Return(&response, nil)
			}

			responder.Respond()

			require.Equal(t, v.response.Code, v.expectedStatusCode)
			require.JSONEq(t, v.expectedResponseBody, v.response.Body.String())

		})
	}

}
