package server

import (
	"github.com/djordjev/webhook-simulator/internal/packages/mapping"
	"log"
	"net/http"
)

type Matcher interface {
	Match()
	IsMatch() bool
}

type RequestMatcher struct {
	request *http.Request
	flow    *mapping.Flow
	body    map[string]any
	isMatch bool
}

func (m *RequestMatcher) Match() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in f", r)
		}
	}()

	flowRequest := m.flow.Request

	// Match method
	if m.request.Method != flowRequest.Method {
		return
	}

	// Match path
	if m.request.URL.Path != flowRequest.Path {
		return
	}

	// Match body
	if !isMatching(m.flow.Request.Body, m.body) {
		return
	}

	// Match headers
	if !m.headersMatching() {
		return
	}

	m.isMatch = true

	return
}

func (m *RequestMatcher) headersMatching() bool {
	headers := m.flow.Request.Headers

	if headers == nil {
		return true
	}

	for k, v := range headers {
		header := m.request.Header.Get(k)
		if header != v {
			return false
		}
	}

	return true
}

func isMatching(needToMatch map[string]any, object map[string]any) bool {
	if needToMatch == nil {
		return true
	}

	for k, v := range needToMatch {
		switch t := v.(type) {
		case map[string]any:
			{
				inRequest, found := object[k]
				if !found {
					return false
				}

				if casted, ok := inRequest.(map[string]any); ok {
					if !isMatching(t, casted) {
						return false
					}
				}
			}

		default:
			{
				inRequest, found := object[k]
				if !found || inRequest != t {
					return false
				}
			}
		}
	}

	return true
}

func (m *RequestMatcher) IsMatch() bool {
	return m.isMatch
}

var RequestMatchBuilder MatchBuilder = func(request *http.Request, flow *mapping.Flow, body map[string]any) Matcher {
	return &RequestMatcher{request: request, flow: flow, body: body}
}

type MatchBuilder func(request *http.Request, flow *mapping.Flow, body map[string]any) Matcher
