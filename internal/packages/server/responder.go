package server

import (
	"fmt"
	"github.com/djordjev/webhook-simulator/internal/packages/mapping"
	"log"
	"maps"
	"net/http"
	"reflect"
	"strings"
)

type RequestResponder interface {
	Match()
	IsMatch() bool
	Respond()
}

type Responder struct {
	request *http.Request
	flow    *mapping.Flow
	body    map[string]any
	isMatch bool
}

func (m Responder) Match() {
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

	m.isMatch = true

	return
}

func (m Responder) Respond() {
	responseBody := make(map[string]any)

	if m.flow.RespondWithRequest {
		maps.Copy(responseBody, m.body)
	}

	m.visitMapNode("", m.flow.Response.Payload)

}

func (m Responder) visitMapNode(prefix string, obj map[string]any) {
	for k, v := range obj {
		kind := reflect.ValueOf(v).Kind()
		isMap := kind == reflect.Map
		isString := kind == reflect.String

		currentKey := fmt.Sprintf("%s.%s", prefix, k)

		if isMap {
			m.visitMapNode(currentKey, v.(map[string]any))
		} else {
			if isString && m.isVariable(v.(string)) {
				obj[k] = m.getFromRequest(v.(string))
			} else {
				obj[k] = v
			}
		}
	}
}

func (m Responder) isVariable(str string) bool {
	return strings.HasPrefix(str, "{{$") && strings.HasSuffix(str, "}}")
}

func (m Responder) getFromRequest(key string) any {
	keyNoPrefix, _ := strings.CutPrefix(key, "{{$")
	keyFinal, _ := strings.CutSuffix(keyNoPrefix, "}}")

	segments := strings.Split(keyFinal, ".")
	for _, v := range segments {
		log.Println(v)
	}

	return "dsa"
}

func (m Responder) IsMatch() bool {
	return m.isMatch
}

var Builder RespondBuilder = func(request *http.Request, flow *mapping.Flow, body map[string]any) RequestResponder {
	return Responder{request: request, flow: flow, body: body}
}

type RespondBuilder func(request *http.Request, flow *mapping.Flow, body map[string]any) RequestResponder
