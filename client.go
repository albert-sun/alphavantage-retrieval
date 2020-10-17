package main

import (
	akitohttp "akito/packages/http"
	"context"
	"github.com/valyala/fasthttp"
	"time"
)

// alphaResult contains the request URL, fields for placing the body and error, and a notification channel.
type alphaResult struct {
	url   string        // request URL
	notif chan struct{} // when done
	body  []byte
	err   error
}

// alphaClient communicates with the Alphavantage API for downloading purposes.
// Mostly implements thread-limiting for preventing server-sided issues.
type alphaClient struct {
	apiKey       string
	limit        int                // max requests per minute
	killCtx      context.Context    // for managing and killing goroutines
	killFunc     context.CancelFunc // ^^
	httpClient   *fasthttp.Client   // shared http client (doesn't exceed connection limit)
	requestsChan chan *alphaResult  // processes function and sends result through channel
}

// newAlphaClient initializes a new alphaClient with the passed API key.
// Also initializes the goroutine for concurrently handling (and thread-limiting) requests.
func newAlphaClient(apiKey string, limit int) *alphaClient {
	ctx, ctxCancel := context.WithCancel(context.Background())

	// initialize client parameters
	client := alphaClient{
		apiKey:       apiKey,
		limit:        limit,
		killCtx:      ctx,                        // actual context being cancelled
		killFunc:     ctxCancel,                  // function for cancelling (killing) goroutines
		httpClient:   &fasthttp.Client{},         // default un-proxied client
		requestsChan: make(chan *alphaResult, 5), // speed issues vs 10?
	}

	// begin perpetual goroutines
	go client.handleRequests()

	return &client
}

// handleRequests concurrently handles requests using the dirty method of maintaining consumer goroutines.
// All underlying goroutines can be cancelled through client.killCtx.
func (client *alphaClient) handleRequests() {
	for {
		select {
		case result := <-client.requestsChan: // received request url
			go func(res *alphaResult) {
				response, err := akitohttp.RequestGET(client.httpClient, res.url) // perform request
				if err != nil {
					res.err = err // keep body == nil
				} else {
					res.body = []byte(string(response.Body())) // deep copy, should work
					fasthttp.ReleaseResponse(response)
				}

				res.notif <- struct{}{} // send notification of completion
			}(result)
		case <-client.killCtx.Done(): // goroutines killed
			return
		}

		// primitive rate limit method: wait constant time
		time.Sleep(time.Duration(60/client.limit) * time.Second)
	}
}
