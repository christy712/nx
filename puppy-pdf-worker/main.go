package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"puppy-pdf-worker/chromeUtils"
	"puppy-pdf-worker/sqsUtils"

	"github.com/chromedp/chromedp"
)

var (
	env              = sqsUtils.LoadEnv()
	sem              = make(chan struct{}, env.MaxThreads)
	browserCtx       context.Context
	browserCtxCancel context.CancelFunc
	browserCtxLock   = &sync.Mutex{}
)

func main() {

	browserCtx, browserCtxCancel = chromeUtils.CreateBrowserContext(browserCtx, browserCtxCancel)
	go func() {

		for {
			time.Sleep(30 * time.Second)
			testctx, testCancel := chromedp.NewContext(browserCtx)
			err := chromedp.Run(testctx,
				chromedp.Navigate("about:blank"),
			)
			testCancel()
			if err != nil {
				log.Println("Detected broken browserCtx. Restarting Chrome...")
				browserCtx, browserCtxCancel = chromeUtils.CreateBrowserContext(browserCtx, browserCtxCancel)
			}

		}

	}()

	runtime.GOMAXPROCS(runtime.NumCPU())
	ctx, cancel := context.WithCancel(context.Background())
	go listenForSignals(cancel)

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully...")
			return
		default:
			fmt.Println("adding.......")
			sem <- struct{}{}

			go func() {
				processQueue()
			}()
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

func processQueue() {

	time.Sleep(10 * time.Second)
	fmt.Println("relesaingg  ...")
	<-sem

}
