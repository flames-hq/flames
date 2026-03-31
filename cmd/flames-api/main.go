// Command flames-api starts the Flames control-plane API server.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flames-hq/flames/api"
	"github.com/flames-hq/flames/provider/queue/memqueue"
	"github.com/flames-hq/flames/provider/state/memstate"
	"github.com/flames-hq/flames/transport/httpapi"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	ss := memstate.New()
	wq := memqueue.New()
	svc := api.New(ss, wq)
	handler := httpapi.NewHandler(svc)

	srv := &http.Server{Addr: *addr, Handler: handler}

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("listening on %s", *addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
