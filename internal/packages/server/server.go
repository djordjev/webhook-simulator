package server

import (
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
	config       config.Config
	mapper       mapping.Mapper
	matchBuilder RespondBuilder
}

func (s server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

		responder := s.matchBuilder(request, &current, body)

		wg.Add(1)

		go func() {
			defer wg.Done()
			responder.Match()

			if !responder.IsMatch() {
				return
			}

			log.Println(fmt.Sprintf("propagating request to %s %s", current.Response.Method, current.Response.Path))
			responder.Respond()
		}()
	}

	wg.Wait()

	writer.WriteHeader(200)
}

func NewServer(cfg config.Config, mapper mapping.Mapper, matchBuilder RespondBuilder) http.Handler {
	srv := server{config: cfg, mapper: mapper, matchBuilder: matchBuilder}

	return srv
}
