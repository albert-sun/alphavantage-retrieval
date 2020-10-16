package main

// alphaParams maps Alphavantage function names to slices of their required parameters.
// Only used in preliminary error handling before request. Index 0 = required, index 1 = optional.
// Does not include apiKey as a parameter because its inclusion is already assumed.
var alphaParams = map[string][][]string{
	// Stock Time Series
	"TIME_SERIES_INTRADAY":          {[]string{"symbol", "interval"}, []string{"adjusted", "outputsize", "datatype"}},
	"TIME_SERIES_INTRADAY_EXTENDED": {[]string{"symbol", "interval", "slice"}, []string{"adjusted"}},
	"TIME_SERIES_DAILY":             {[]string{"symbol"}, []string{"outputsize", "datatype"}},
	"TIME_SERIES_DAILY_ADJUSTED":    {[]string{"symbol"}, []string{"outputsize", "datatype"}},
	"TIME_SERIES_WEEKLY":            {[]string{"symbol"}, []string{"outputsize", "datatype"}},
	"TIME_SERIES_WEEKLY_ADJUSTED":   {[]string{"symbol"}, []string{"outputsize", "datatype"}},
	"TIME_SERIES_MONTHLY":           {[]string{"symbol"}, []string{"outputsize", "datatype"}},
	"TIME_SERIES_MONTHLY_ADJUSTED":  {[]string{"symbol"}, []string{"outputsize", "datatype"}},
	"GLOBAL_QUOTE":                  {[]string{"symbol"}, []string{"outputsize", "datatype"}},
	"SYMBOL_SEARCH":                 {[]string{"keywords"}, []string{"datatype"}},
}

// download provides the basic foundation for retrieving data from Alphavantage.
func (client *alphaClient) download(url string) ([]byte, error) {
	result := alphaResult{
		url:   url,
		notif: make(chan struct{}),
		body:  nil, // set by handler
		err:   nil, // set by handler
	}

	client.requestsChan <- &result // send to request queue
	<-result.notif                 // wait for completion

	// handle whether request errored
	if result.err != nil { // error
		return nil, result.err
	} else { // no error
		return result.body, nil
	}
}
