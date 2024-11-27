package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/djordjev/webhook-simulator/internal/packages/config"
	"github.com/djordjev/webhook-simulator/internal/packages/mapping"
	"log"
	"maps"
	"net/http"
	"sync"
	"sync/atomic"
)

type server struct {
	config          config.Config
	mapper          mapping.Mapper
	matchBuilder    MatchBuilder
	responseBuilder ResponseBuilder
	appCtx          context.Context
}

func (s server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method == "GET" && request.URL.Path == "/ping" {
		_, _ = writer.Write([]byte("PONG"))
		return
	}

	if s.config.SkipFSEvents {
		err := s.mapper.Refresh()
		if err != nil {
			log.Println("unable to read configuration")
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	payload := make(map[string]any)

	err := json.NewDecoder(request.Body).Decode(&payload)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	mappings := s.mapper.GetMappings()
	if len(mappings) == 0 {
		writer.WriteHeader(http.StatusNoContent)
		return
	}

	var wg sync.WaitGroup
	var counter atomic.Int32

	for _, current := range mappings {
		body := make(map[string]any)
		maps.Copy(body, payload)

		matcher := s.matchBuilder(request, &current, body)
		wg.Add(1)

		go func() {
			defer wg.Done()
			matcher.Match()

			if !matcher.IsMatch() {
				return
			}

			if res := counter.Swap(1); res > 0 {
				log.Println("multiple matchers are matching this request. Ignoring...")
				return
			}

			log.Println(fmt.Sprintf("request matched %s %s", current.Request.Method, current.Request.Path))

			responder := s.responseBuilder(
				request,
				&current,
				body,
				writer,
				s.appCtx,
				http.DefaultClient,
			)

			responder.Respond()

		}()
	}

	wg.Wait()

	if counter.Load() == 0 {
		writer.WriteHeader(http.StatusBadRequest)
	}
}

func NewServer(
	cfg config.Config,
	mapper mapping.Mapper,
	matchBuilder MatchBuilder,
	responseBuilder ResponseBuilder,
	appCtx context.Context,
) http.Handler {
	srv := server{
		config:          cfg,
		mapper:          mapper,
		matchBuilder:    matchBuilder,
		responseBuilder: responseBuilder,
		appCtx:          appCtx,
	}

	return srv
}
