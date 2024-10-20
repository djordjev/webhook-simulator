package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/djordjev/webhook-simulator/internal/packages/mapping"
	"io"
	"log"
	"maps"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Responder interface {
	Respond()
}

type RequestResponder struct {
	request *http.Request
	flow    *mapping.Flow
	body    map[string]any
	rw      http.ResponseWriter
	mainCtx context.Context
}

func (r RequestResponder) Respond() {
	reqDelay := time.Duration(r.flow.Response.Delay)
	webhookDelay := time.Duration(r.flow.WebHook.Delay)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-time.After(reqDelay * time.Millisecond):
			{
				r.respondHttp()
			}

		case <-r.mainCtx.Done():
			{
				log.Println("canceling timeout for response")
				return
			}
		}
	}()

	if r.flow.WebHook != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-time.After(webhookDelay * time.Millisecond):
				{
					r.triggerWebHook()
				}

			case <-r.mainCtx.Done():
				{
					log.Println("canceling timeout for webhook")
					return
				}
			}
		}()
	}

	wg.Wait()

}

func (r RequestResponder) respondHttp() {
	payload := r.constructPayload(
		r.flow.Response.IncludeRequest,
		r.flow.Response.Body,
	)

	for k, v := range r.flow.Response.Headers {
		replaced, _ := r.replaceValue(v)
		if strReplaced, ok := replaced.(string); ok {
			r.rw.Header().Set(k, strReplaced)
		}
	}

	code := r.flow.Response.Code
	if r.flow.Response.Code == 0 {
		code = http.StatusOK
	}

	r.rw.WriteHeader(code)
	_, err := r.rw.Write(payload)
	if err != nil {
		log.Println("unable to send a response")
	}
}

func (r RequestResponder) triggerWebHook() {
	payload := r.constructPayload(
		r.flow.WebHook.IncludeRequest,
		r.flow.WebHook.Body,
	)

	for k, v := range r.flow.WebHook.Headers {
		replaced, _ := r.replaceValue(v)
		if strReplaced, ok := replaced.(string); ok {
			r.rw.Header().Set(k, strReplaced)
		}
	}

	log.Println("sending request" + string(payload))

	body := bytes.NewReader(payload)

	req, err := http.NewRequest(r.flow.WebHook.Method, r.flow.WebHook.Path, body)
	if err != nil {
		log.Println("unable to create request for webhook")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil || res == nil {
		log.Println("error while receiving webhook response")
		return
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println("unable to read webhook response body")
		return
	}

	log.Println("received response from webhook" + string(resBody))
}

func (r RequestResponder) constructPayload(includeRequest bool, data map[string]any) []byte {
	response := make(map[string]any)

	if includeRequest {
		maps.Copy(response, r.body)
	}

	err := r.mergeInto(response, data)
	if err != nil {
		log.Println("unable to replace variables")
		return []byte("")
	}

	marshalled, err := json.Marshal(response)
	if err != nil {
		log.Println("unable to marshal", response)
	}

	return marshalled
}

func (r RequestResponder) mergeInto(dst map[string]any, source map[string]any) error {
	for k, v := range source {
		// If not found just add the one from source
		innerVal, found := dst[k]
		if !found {
			dst[k] = v
			innerVal = v
		}

		switch value := v.(type) {
		case map[string]any:
			{
				// found and not map - just overwrite
				casted, ok := innerVal.(map[string]any)
				if !ok {
					dst[k] = value
				}

				// found and it's map - do recursively
				err := r.mergeInto(casted, value)
				if err != nil {
					return err
				}
			}

		default:
			{

				if strVal, ok := v.(string); ok {
					replacedValue, err := r.replaceValue(strVal)

					if err != nil {
						return err
					}

					dst[k] = replacedValue
				} else {
					dst[k] = v
				}
			}

		}
	}

	return nil
}

func (r RequestResponder) replaceValue(valuePlaceholder string) (any, error) {
	if !strings.HasPrefix(valuePlaceholder, "${{") || !strings.HasSuffix(valuePlaceholder, "}}") {
		return valuePlaceholder, nil
	}

	noSuffix, _ := strings.CutSuffix(valuePlaceholder, "}}")
	value, _ := strings.CutPrefix(noSuffix, "${{")

	var err error
	var returnValue any
	if strings.HasPrefix(value, "body") {
		noBody, found := strings.CutPrefix(value, "body.")
		if !found {
			return "", errors.New("unable to find path" + valuePlaceholder)
		}
		returnValue, err = r.getFromBody(noBody)
	} else if strings.HasPrefix(value, "header") {
		noHeader, found := strings.CutPrefix(value, "header.")
		if !found {
			return "", errors.New("unable to find path" + valuePlaceholder)
		}
		returnValue, err = r.getFromHeader(noHeader)
	}

	if err != nil {
		return "", err
	}

	return returnValue, nil
}

func (r RequestResponder) getFromBody(value string) (any, error) {
	segments := strings.Split(value, ".")

	current := r.body
	length := len(segments)

	for k, v := range segments {
		isLast := k == length-1

		currentVal, found := current[v]

		if !found {
			return "", fmt.Errorf("unable to find segment %s in path %s", v, value)
		}

		if isLast {
			return currentVal, nil
		} else {
			if next, ok := currentVal.(map[string]any); ok {
				current = next
			} else {
				return "", errors.New("next value is not a map" + value)

			}

		}

	}

	return "", errors.New("not found for path" + value)
}

func (r RequestResponder) getFromHeader(value string) (any, error) {
	val := r.request.Header.Get(value)

	if val == "" {
		return "", errors.New("cant find in header " + value)
	}

	return val, nil
}

var RequestResponseBuilder ResponseBuilder = func(
	request *http.Request,
	flow *mapping.Flow,
	body map[string]any,
	rw http.ResponseWriter,
	mainCtx context.Context,
) Responder {
	return RequestResponder{
		request: request,
		flow:    flow,
		body:    body,
		rw:      rw,
		mainCtx: mainCtx,
	}
}

type ResponseBuilder func(request *http.Request, flow *mapping.Flow, body map[string]any, rw http.ResponseWriter, mainCtx context.Context) Responder
