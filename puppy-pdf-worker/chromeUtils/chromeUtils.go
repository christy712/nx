package chromeUtils

import (
	"context"
	"fmt"
	"sync"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type RequestPayload struct {
	URL string `json:"url"`
}
type PdfError struct {
	Error error
}

var browserCtxCancel context.CancelFunc
var browserCtxLock = &sync.Mutex{}

func handlePDF(URL string, browserCtx context.Context) {

	errChan := make(chan PdfError, 1)

	ctx, BrowserCancel := chromedp.NewContext(browserCtx)
	defer BrowserCancel()
	var buf []byte
	go func() {
		if err := chromedp.Run(ctx, printToPDF(URL, &buf)); err != nil {
			fmt.Println(err)
			errChan <- PdfError{Error: err}
		} else {
			errChan <- PdfError{Error: nil}
		}

	}()

	select {

	case v := <-errChan:
		if v.Error != nil {
			return
		}
		fmt.Println("sucessfully generated pdf")
		OutputResponse(&buf)
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

func OutputResponse(buf *[]byte) {

	// output_file_name := time.Now().Format("03_04_05") + ".pdf"
	// //output_file := "/usr/src/app/output/" + output_file_name
	// response := make(map[string]string)
	// // if err := os.WriteFile(output_file, *buf, 0o644); err != nil {
	// // 	//log.Fatal(err)
	// // 	http.Error(w, "Failed to save file ", http.StatusBadRequest)
	// // 	return
	// // }

	// url, err := s3utils.PutObject(env.Bucket, "pdf/"+output_file_name, bytes.NewReader(*buf), env.S3Region)

	// fmt.Println(env.S3Region)
	// if err != nil {
	// 	response["message"] = "PDF saved failed"
	// 	response["error"] = err.Error()

	// } else {
	// 	response["message"] = "PDF saved successfully"
	// 	response["url"] = url
	// }

}

func CreateBrowserContext(browserCtx context.Context, browserCtxCancel context.CancelFunc) (context.Context, context.CancelFunc) {

	fmt.Print("1")
	browserCtxLock.Lock()
	defer browserCtxLock.Unlock()

	if browserCtxCancel != nil {
		browserCtxCancel()
	}
	fmt.Print("2")

	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
	)
	fmt.Print("3")
	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), options...)
	browserCtx, browserCtxCancel = chromedp.NewContext(allocCtx)
	return browserCtx, browserCtxCancel

}
