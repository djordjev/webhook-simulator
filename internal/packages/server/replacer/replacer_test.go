package replacer

import (
	"bytes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestReplace(t *testing.T) {
	now = func() time.Time {
		return time.Date(2024, 10, 27, 20, 34, 58, 0, time.UTC)
	}

	uuidToReturn := uuid.New()

	newUUID = func() uuid.UUID { return uuidToReturn }

	testCases := []struct {
		name     string
		body     map[string]any
		headers  map[string]string
		input    string
		result   any
		iterator any
	}{
		{
			name:    "no replacement if no variable",
			body:    map[string]any{},
			headers: map[string]string{},
			input:   "no variables",
			result:  "no variables",
		},
		{
			name:    "replace whole string",
			body:    map[string]any{"name": "Jon"},
			headers: map[string]string{},
			input:   "${{body.name}}",
			result:  "Jon",
		},
		{
			name:    "replaces whole number",
			body:    map[string]any{"age": 35},
			headers: map[string]string{},
			input:   "${{body.age}}",
			result:  35,
		},
		{
			name:    "replaces from header",
			body:    map[string]any{},
			headers: map[string]string{"Content-Type": "application/json"},
			input:   "${{header.Content-Type}}",
			result:  "application/json",
		},
		{
			name:    "replaces two strings",
			body:    map[string]any{"name": "Jon"},
			headers: map[string]string{"Content-Type": "application/json"},
			input:   "user ${{body.name}} with contentType ${{header.Content-Type}}",
			result:  "user Jon with contentType application/json",
		},
		{
			name:    "replaces string and number",
			body:    map[string]any{"name": "Jon", "age": 35},
			headers: map[string]string{},
			input:   "${{body.name}} is ${{body.age}} years old",
			result:  "Jon is 35 years old",
		},
		{
			name:    "returns current date",
			body:    map[string]any{},
			headers: map[string]string{},
			input:   "${{now}}",
			result:  "2024-10-27T20:34:58Z",
		},
		{
			name:    "returns time after",
			body:    map[string]any{},
			headers: map[string]string{},
			input:   "${{after 2 seconds}}",
			result:  "2024-10-27T20:35:00Z",
		},
		{
			name:    "returns time before",
			body:    map[string]any{},
			headers: map[string]string{},
			input:   "${{before 1 days}}",
			result:  "2024-10-26T20:34:58Z",
		},
		{
			name:    "returns UUID",
			body:    map[string]any{},
			headers: map[string]string{},
			input:   "${{uuid}}",
			result:  uuidToReturn.String(),
		},
		{
			name:     "picks up value from iterator - object",
			body:     map[string]any{},
			headers:  map[string]string{},
			iterator: map[string]any{"value": "randomValue"},
			input:    "${{iterator.value}}",
			result:   "randomValue",
		},
		{
			name:     "picks up value from iterator - whole iterator",
			body:     map[string]any{},
			headers:  map[string]string{},
			iterator: "whole",
			input:    "${{iterator.}}",
			result:   "whole",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "", bytes.NewBufferString(""))

			for headerKey, headerValue := range test.headers {
				req.Header.Set(headerKey, headerValue)
			}

			replacer := stringReplacer{
				body:     test.body,
				header:   req.Header,
				iterator: test.iterator,
			}

			result, _ := replacer.Replace(test.input)

			require.Equal(t, test.result, result)
		})
	}

}
