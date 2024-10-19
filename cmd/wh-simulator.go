package main

import (
	"errors"
	"fmt"
	"github.com/djordjev/webhook-simulator/internal/packages/config"
	"github.com/djordjev/webhook-simulator/internal/packages/mapping"
	"github.com/djordjev/webhook-simulator/internal/packages/server"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	cfg := config.ParseConfig()
	mapper := mapping.NewMapping(cfg, os.DirFS(cfg.Mapping))

	srv := server.NewServer(cfg, mapper, server.Builder)

	err := mapper.Refresh()

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
