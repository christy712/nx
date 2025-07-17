package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	go listenForSignals(cancel)

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully...")
			return
		default:
			// Poll and process
		}
	}

}
func listenForSignals(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)

	// Listen for SIGINT (Ctrl+C) and SIGTERM (Docker stop)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-sigChan
	log.Printf("Received signal: %s. Initiating shutdown...", sig)

	// Trigger the context cancellation
	cancel()
}
