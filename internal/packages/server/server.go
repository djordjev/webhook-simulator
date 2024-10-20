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

			log.Println(fmt.Sprintf("request matched %s %s", current.Request.Method, current.Request.Path))

			responder := s.responseBuilder(request, &current, body, writer, s.appCtx)
			responder.Respond()

		}()
	}

	wg.Wait()

	writer.WriteHeader(200)
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
