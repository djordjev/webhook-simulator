package main

import (
	"errors"
	"fmt"
	"github.com/djordjev/webhook-simulator/internal/packages/config"
	"github.com/djordjev/webhook-simulator/internal/packages/mapping"
	"github.com/djordjev/webhook-simulator/internal/packages/server"
	"github.com/djordjev/webhook-simulator/internal/packages/updating"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	cfg := config.ParseConfig()
	fs := os.DirFS(cfg.Mapping)

	mapper := mapping.NewMapping(cfg, fs)

	srv := server.NewServer(
		cfg, mapper,
		server.RequestMatchBuilder,
		server.RequestResponseBuilder,
	)

	err := mapper.Refresh()

	listener := updating.NewUpdater(mapper, cfg)
	listener.Listen()

	if err != nil {
		log.Fatalf("unable to read files from mapping directory")
	}

	port := cfg.Port

	log.Printf("opening server on port %d\n", port)

	err = http.ListenAndServe(fmt.Sprintf(":%d", port), srv)
	if errors.Is(err, net.ErrClosed) {
		log.Println("server closed")
	} else if err != nil {
		log.Println("error opening server")
	}
}
