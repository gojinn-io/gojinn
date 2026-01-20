package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 1. Configuration via Flags (CLI Guidelines)
	// Allow the user to change the port without recompiling
	port := flag.String("port", "9090", "TCP port to listen on")
	flag.Parse()

	addr := ":" + *port

	// 2. Setup Multiplexer
	mux := http.NewServeMux()

	// Handler: Catch-All (Legacy Monolith simulation)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Log request to Stderr (Operational Log)
		log.Printf("[Legacy] Serving page '%s' (Monolith)\n", r.URL.Path)
		fmt.Fprintf(w, "👴 Legacy Monolith: Serving page '%s' (PHP/Java/Python simulation)", r.URL.Path)
	})

	// Handler: Slow Endpoint (Bottleneck simulation)
	mux.HandleFunc("/api/calc", func(w http.ResponseWriter, r *http.Request) {
		log.Println("[Legacy] Receiving heavy calculation request...")

		// Simulate latency
		time.Sleep(2 * time.Second)

		fmt.Fprintf(w, "👴 Legacy Result: Heavy calculation done in 2s (Too slow!)")
	})

	// 3. Server Configuration
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// 4. Run Server in a Goroutine (Non-blocking)
	go func() {
		log.Printf("👴 Legacy Server starting on port %s...\n", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 5. Graceful Shutdown (CLI Guidelines)
	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("👴 Legacy Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("👴 Legacy Server exited properly")
}
