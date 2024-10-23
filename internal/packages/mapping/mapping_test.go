package mapping

import (
	"github.com/djordjev/webhook-simulator/internal/packages/config"
	"github.com/stretchr/testify/require"
	"io/fs"
	"testing"
	"testing/fstest"
)

type dataPair struct {
	json string
	flow Flow
}

var firstPair = dataPair{
	json: `
		{
			"request": {
				"method": "POST",
				"path": "/testPath1",
				"body": { "user": { "username": "test_username"}}
			},
			"response": {
				"delay": 300,
				"includeRequest": true,
				"code": 200,
				"headers": {},
				"body": { "random": "response" }
			},
			"web_hook": {
				"method": "GET",
				"path": "www.google.com",
				"delay": 20,
				"includeRequest": false,
				"headers": {},
				"body": { "send_to": "web_hook" }
			}
		}
	`,
	flow: Flow{
		Request: &RequestDefinition{
			Method: "POST",
			Path:   "/testPath1",
			Body:   map[string]any{"user": map[string]any{"username": "test_username"}},
		},
		Response: &ResponseDefinition{
			Code:           200,
			Delay:          300,
			IncludeRequest: true,
			Headers:        map[string]string{},
			Body:           map[string]any{"random": "response"},
		},
		WebHook: &WebHookDefinition{
			Method:         "GET",
			Path:           "www.google.com",
			Delay:          20,
			IncludeRequest: false,
			Headers:        map[string]string{},
			Body:           map[string]any{"send_to": "web_hook"},
		},
	},
}

var secondPair = dataPair{
	json: `
		{
			"request": {
				"method": "PUT",
				"path": "/testPath2",
				"body": { "user": { "role": "admin"}}
			},
			"response": {
				"delay": 0,
				"includeRequest": true,
				"code": 400,
				"headers": {
					"x-api-key": "123"
				},
				"body": { "error": "message" }
			}
		}
	`,
	flow: Flow{
		Request: &RequestDefinition{
			Method: "PUT",
			Path:   "/testPath2",
			Body:   map[string]any{"user": map[string]any{"role": "admin"}},
		},
		Response: &ResponseDefinition{
			Code:           400,
			IncludeRequest: true,
			Headers:        map[string]string{"x-api-key": "123"},
			Body:           map[string]any{"error": "message"},
		},
	},
}

func TestRefresh(t *testing.T) {
	testCases := []struct {
		name   string
		fs     fs.FS
		result []Flow
	}{
		{
			name:   "empty directory",
			fs:     fstest.MapFS{},
			result: []Flow{},
		},
		{
			name: "directory with one file",
			fs: fstest.MapFS{
				"file1.whs": {Data: []byte(firstPair.json)},
			},
			result: []Flow{firstPair.flow},
		},
		{
			name: "does not read files that are not with .whs extension",
			fs: fstest.MapFS{
				"file1.whs":     {Data: []byte(firstPair.json)},
				"file2.not-whs": {Data: []byte(secondPair.json)},
			},
			result: []Flow{firstPair.flow},
		},
		{
			name: "ignores non parsable files",
			fs: fstest.MapFS{
				"file1.whs": {Data: []byte(firstPair.json)},
				"file2.whs": {Data: []byte("{ wrong: json")},
			},
			result: []Flow{firstPair.flow},
		},
		{
			name: "reads two correct files",
			fs: fstest.MapFS{
				"file1.whs": {Data: []byte(firstPair.json)},
				"file2.whs": {Data: []byte(secondPair.json)},
			},
			result: []Flow{firstPair.flow, secondPair.flow},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			testMapping := NewMapping(config.Config{}, test.fs)

			_ = testMapping.Refresh()

			require.ElementsMatch(t, testMapping.GetMappings(), test.result)
		})
	}

}
