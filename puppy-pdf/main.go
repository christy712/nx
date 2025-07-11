package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"puppy-pdf/s3utils"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type RequestPayload struct {
	URL string `json:"url"`
}
type PdfError struct {
	Error error
}
type EnvVars struct {
	Bucket     string
	Region     string
	MaxThreads int
}

var env = loadEnv()

var browserCtx context.Context
var browserCtxCancel context.CancelFunc
var browserCtxLock = &sync.Mutex{}

// var sem = make(chan struct{}, runtime.NumCPU())
var sem = make(chan struct{}, env.MaxThreads)

func handlePDF(w http.ResponseWriter, r *http.Request) {
	var req RequestPayload
	fmt.Println("this request......")
	fmt.Println(r.Body)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	sem <- struct{}{}
	defer func() {
		<-sem
	}()

	requestCtx := r.Context()
	errChan := make(chan PdfError, 1)

	ctx, BrowserCancel := chromedp.NewContext(browserCtx)
	defer BrowserCancel()
	var buf []byte
	go func() {
		if err := chromedp.Run(ctx, printToPDF(req.URL, &buf)); err != nil {
			errChan <- PdfError{Error: err}
		} else {
			errChan <- PdfError{Error: nil}
		}

	}()

	select {

	case v := <-errChan:

		if v.Error != nil {
			http.Error(w, "Failed to run chrome", http.StatusBadRequest)
			return
		}
		fmt.Println("sucessfully generated pdf")
		OutputResponse(w, &buf)

	case <-requestCtx.Done():
		BrowserCancel()
		fmt.Println("cancelled by cliennt ")
		return

	}

}

// print a specific pdf page.
func printToPDF(urlstr string, res *[]byte) chromedp.Tasks {
	fmt.Println("generating......")
	return chromedp.Tasks{
		chromedp.Navigate(urlstr),
		chromedp.Evaluate(`new Promise(resolve => {
        let timeout;
        const observer = new MutationObserver(() => {
            clearTimeout(timeout);
            timeout = setTimeout(() => {
                observer.disconnect();
                resolve(true);
            }, 500);
        });
        observer.observe(document.body, { childList: true, subtree: true, attributes: true });
        timeout = setTimeout(() => {
            observer.disconnect();
            resolve(true);
        }, 3000);
    })`, nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithDisplayHeaderFooter(true).
				WithFooterTemplate(`
                <div style="font-size:10px; width:100%; text-align:center;">
                    Page <span class="pageNumber"></span> of <span class="totalPages"></span>
                </div>
            `).Do(ctx)
			if err != nil {
				return err
			}
			*res = buf
			return nil
		}),
	}
}

func OutputResponse(w http.ResponseWriter, buf *[]byte) {

	output_file_name := time.Now().Format("03_04_05") + ".pdf"
	//output_file := "/usr/src/app/output/" + output_file_name
	response := make(map[string]string)
	// if err := os.WriteFile(output_file, *buf, 0o644); err != nil {
	// 	//log.Fatal(err)
	// 	http.Error(w, "Failed to save file ", http.StatusBadRequest)
	// 	return
	// }

	url, err := s3utils.PutObject(env.Bucket, "pdf/"+output_file_name, bytes.NewReader(*buf), env.Region)

	fmt.Println(env.Region)
	if err != nil {
		response["message"] = "PDF saved failed"
		response["error"] = err.Error()

	} else {
		response["message"] = "PDF saved successfully"
		response["url"] = url
	}

	res, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func CreateBrowserContext() {

	browserCtxLock.Lock()
	defer browserCtxLock.Unlock()

	if browserCtxCancel != nil {
		browserCtxCancel()
	}

	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
	)
	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), options...)
	browserCtx, browserCtxCancel = chromedp.NewContext(allocCtx)

}

func loadEnv() EnvVars {

	file, err := os.Open(".cutsom.env")
	if err != nil {

		log.Fatal("No .custom.env Found ")
		os.Exit(200)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	env_map := map[string]string{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines or comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Split line into key and value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		env_map[key] = value

	}

	//fmt.Println(env_map)
	mt, err := strconv.Atoi(env_map["CPU_MAX_THREADS"])
	if err != nil || mt <= 0 {
		mt = runtime.NumCPU()
	}
	return EnvVars{
		Bucket:     env_map["S3_BUCKET"],
		Region:     env_map["S3_REGION"],
		MaxThreads: mt,
	}
}

func main() {
	fmt.Println(env)
	CreateBrowserContext()
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
				CreateBrowserContext()
			}

		}

	}()

	runtime.GOMAXPROCS(runtime.NumCPU())
	http.HandleFunc("/generate", handlePDF)

	port := "8080"
	fmt.Println("Listening on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
