package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/djordjev/webhook-simulator/internal/packages/mapping"
	"github.com/djordjev/webhook-simulator/internal/packages/server/replacer"
	"io"
	"log"
	"maps"
	"net/http"
	"reflect"
	"sync"
	"time"
)

const Each = "$each"
const Field = "$field"
const To = "$to"

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Responder interface {
	Respond()
}

type RequestResponder struct {
	request    *http.Request
	flow       *mapping.Flow
	body       map[string]any
	rw         http.ResponseWriter
	mainCtx    context.Context
	httpClient HTTPClient
	replacer   replacer.Replacer
}

func (r RequestResponder) Respond() {
	reqDelay := time.Duration(r.flow.Response.Delay)

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
		webhookDelay := time.Duration(r.flow.WebHook.Delay)

		go func() {

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
		replaced, _ := r.replacer.Replace(v)
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
		replaced, _ := r.replacer.Replace(v)
		if strReplaced, ok := replaced.(string); ok {
			r.rw.Header().Set(k, strReplaced)
		}
	}

	log.Println("sending webhook request" + string(payload))

	body := bytes.NewReader(payload)

	req, err := http.NewRequest(r.flow.WebHook.Method, r.flow.WebHook.Path, body)

	if err != nil {
		log.Println("unable to create request for webhook")
	}

	res, err := r.httpClient.Do(req)
	if err != nil || res == nil {
		log.Println("error while receiving webhook response", err)
		return
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println("unable to read webhook response body", res.StatusCode)
		return
	}

	log.Println("received response from webhook", "status code", res.StatusCode, string(resBody))
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

func (r RequestResponder) mustMapStringAny(unknown any) map[string]any {
	result := make(map[string]any)

	if mapStrings, ok := unknown.(map[string]any); ok {
		return mapStrings
	}

	if anyString, ok := unknown.(map[any]any); ok {
		for k, v := range anyString {
			if str, strOK := k.(string); strOK {
				result[str] = v
			}
		}
	}

	return result
}

func (r RequestResponder) isArrayMapper(m map[string]any) bool {
	for k, _ := range m {
		if k == Each {
			return true
		}
	}

	return false
}

func (r RequestResponder) populateMappedArray(descriptor map[string]any) (result []any) {
	initialReplacer := r.replacer
	defer func() {
		r.replacer = initialReplacer
	}()

	result = make([]any, 0)

	anyField, ok := descriptor[Field]
	if !ok {
		return
	}

	anyTo, ok := descriptor[To]
	if !ok {
		return
	}

	strField, ok := anyField.(string)
	if !ok {
		return
	}

	field, err := r.replacer.Replace(strField)
	if err != nil {
		return
	}

	arrField, ok := field.([]any)
	if !ok {
		return
	}

	for _, v := range arrField {
		r.replacer = r.replacer.Child(v)

		if reflect.TypeOf(anyTo).Kind() == reflect.Map {
			current := make(map[string]any)
			err = r.mergeInto(current, r.mustMapStringAny(anyTo))
			if err != nil {
				log.Println(err)
				return
			}

			result = append(result, current)
		} else if reflect.TypeOf(anyTo).Kind() == reflect.Slice {
			log.Println("can't array map to another array")
			return
		} else {
			str, ok := anyTo.(string)
			if !ok {
				result = append(result, anyTo)
			} else {
				replaced, err := r.replacer.Replace(str)
				if err != nil {
					return
				}
				result = append(result, replaced)
			}
		}
	}

	return result
}

func (r RequestResponder) mergeInto(dst map[string]any, source map[string]any) error {
	for k, v := range source {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Map:
			{
				casted := r.mustMapStringAny(v)

				isArrayMapper := r.isArrayMapper(casted)

				valueDest, found := dst[k]

				if isArrayMapper {
					descriptor, ok := casted[Each]
					if ok {
						dst[k] = r.populateMappedArray(r.mustMapStringAny(descriptor))
					}
				} else if !found || reflect.ValueOf(valueDest).Kind() != reflect.Map {
					empty := map[string]any{}
					dst[k] = empty

					err := r.mergeInto(empty, casted)
					if err != nil {
						return err
					}

				} else {
					destMap := r.mustMapStringAny(valueDest)

					err := r.mergeInto(destMap, casted)
					if err != nil {
						return err
					}
				}
			}

		case reflect.Slice:
			{
				casted, ok := v.([]any)
				if !ok {
					break
				}

				dstSlice, found := dst[k]
				dstSliceCasted, isSlice := dstSlice.([]any)

				if !found || !isSlice {
					dst[k] = r.mergeArrays([]any{}, casted)
				} else {
					dst[k] = r.mergeArrays(dstSliceCasted, casted)
				}

			}

		default:
			{
				if strVal, ok := v.(string); ok {
					replacedValue, err := r.replacer.Replace(strVal)

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

func (r RequestResponder) mergeArrays(dst []any, src []any) []any {
	result := make([]any, 0)

	for index, v := range src {
		var currentDst any = nil
		if index < len(dst) {
			currentDst = dst[index]
		}

		switch reflect.TypeOf(v).Kind() {
		case reflect.Map:
			{
				dstMap, ok := currentDst.(map[string]any)
				if !ok {
					dstMap = map[string]any{}
				}

				if srcMap, ok := v.(map[string]any); ok {
					_ = r.mergeInto(dstMap, srcMap)
				}

				result = append(result, dstMap)
			}

		case reflect.Slice:
			{
				dstSlice, ok := currentDst.([]any)
				if !ok {
					dstSlice = make([]any, 0)
				}

				var res any = make([]any, 0)
				if srcSlice, ok := v.([]any); ok {
					res = r.mergeArrays(dstSlice, srcSlice)
				}

				result = append(result, res)
			}

		default:
			{
				if strVal, ok := v.(string); ok {
					replacedValue, err := r.replacer.Replace(strVal)

					if err == nil {
						result = append(result, replacedValue)
					}
				} else {
					result = append(result, v)
				}
			}

		}
	}

	return result
}

var RequestResponseBuilder ResponseBuilder = func(
	request *http.Request,
	flow *mapping.Flow,
	body map[string]any,
	rw http.ResponseWriter,
	mainCtx context.Context,
	httpClient HTTPClient,
) Responder {
	return RequestResponder{
		request:    request,
		flow:       flow,
		body:       body,
		rw:         rw,
		mainCtx:    mainCtx,
		httpClient: httpClient,
		replacer:   replacer.NewReplacer(body, request.Header),
	}
}

type ResponseBuilder func(
	request *http.Request,
	flow *mapping.Flow,
	body map[string]any,
	rw http.ResponseWriter,
	mainCtx context.Context,
	client HTTPClient,
) Responder
