package main

import (
	akitohttp "akito/packages/http"
	"context"
	"github.com/valyala/fasthttp"
	"sync"
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
	limit        int                // max download threads
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
	wg := sync.WaitGroup{}              // for managing underlying
	for i := 0; i < client.limit; i++ { // create goroutines
		wg.Add(1)
		go func() { // actual perpetual request handler
			defer wg.Done()

			for {
				select {
				case result := <-client.requestsChan: // received request url
					response, err := akitohttp.RequestGET(client.httpClient, result.url) // perform request
					if err != nil {
						result.err = err // keep body == nil
					} else {
						result.body = []byte(string(response.Body())) // deep copy, should work
					}

					result.notif <- struct{}{} // send notification of completion
					fasthttp.ReleaseResponse(response)
				case <-client.killCtx.Done(): // goroutines killed
					return
				}
			}
		}()
	}
}
