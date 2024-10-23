package main

import (
	"context"
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
	"os/signal"
	"syscall"
	"time"
)

func main() {
	mainCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()

	cfg := config.ParseConfig()
	fs := os.DirFS(cfg.Mapping)

	mapper := mapping.NewMapping(cfg, fs)

	srv := server.NewServer(
		cfg,
		mapper,
		server.RequestMatchBuilder,
		server.RequestResponseBuilder,
		mainCtx,
	)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: srv,
	}

	err := mapper.Refresh()

	listener := updating.NewUpdater(mapper, cfg, mainCtx)
	listener.Listen()

	if err != nil {
		log.Fatalf("unable to read files from mapping directory")
	}

	go func() {
		port := cfg.Port
		log.Printf("opening server on port %d\n", port)

		err = httpServer.ListenAndServe()
		if errors.Is(err, net.ErrClosed) {
			log.Println("server closed")
		} else if err != nil {
			log.Println("error opening server")
		}
	}()

	<-mainCtx.Done()

	log.Println("shutting down the server gracefully")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelShutdown()

	err = httpServer.Shutdown(shutdownCtx)
	stop()
	if err != nil {
		log.Println("shutdown has failed")
	}

	log.Println("server stopped")
	os.Exit(0)
}
