package replacer

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestReplace(t *testing.T) {

	testCases := []struct {
		name    string
		body    map[string]any
		headers map[string]string
		input   string
		result  string
	}{
		{
			name:    "replace whole string",
			body:    map[string]any{"name": "Jon"},
			headers: map[string]string{},
			input:   "ASD ${{body.name}} DDD",
			result:  "Jon",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "", bytes.NewBufferString(""))

			for headerKey, headerValue := range test.headers {
				req.Header.Set(headerKey, headerValue)
			}

			replacer := stringReplacer{
				body:   test.body,
				header: req.Header,
			}

			result, _ := replacer.Replace(test.input)

			require.Equal(t, test.result, result)
		})
	}

}
