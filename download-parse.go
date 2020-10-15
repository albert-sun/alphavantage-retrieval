package main

import (
	akitohttp "akito/packages/http"
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/valyala/fasthttp"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

const dateFormat = "2006-01-02 15:04:05" // for formatting date from CSV
var openTime = dayTime{Hour: 9, Minute: 31}

// closeTime found manually since markets sometimes close early

// TODO quite memory inefficient because of large sets of data
// downloadIntradayExt automates concurrently downloading the entirety of intraday data (year1month1 to year2month12).
// Returns a CSV-formatted [][]byte with indices representing month difference form current - for example, the
// previous month's data would have index 1 while the current month's data would have index 0.
// If an error is encountered, returns a nil slice of string-converted response bodies and the appropriate error.
func downloadIntradayExt(apiKey string, symbol string) [][]byte {
	fmt.Printf("Downloading intraday extended trading data for %s ...\n", symbol)

	downloaded := make([][]byte, 24)

	// sequentially download to reduce cpu load?
	httpClient := fasthttp.Client{}    // default client
	wg := sync.WaitGroup{}             // for goroutines
	for year := 1; year <= 2; year++ { // iterate over years
		for month := 1; month <= 12; month++ { // iterate over months (1-indexed)
			wg.Add(1)
			go func(y int, m int) {
				defer wg.Done()

				index := (y-1)*12 + m - 1                       // for slice
				yearMonth := fmt.Sprintf("year%dmonth%d", y, m) // create string
				downloadURL := fmt.Sprintf(
					"https://www.alphavantage.co/query?function=%s&symbol=%s&interval=%s&slice=%s&apikey=%s",
					"TIME_SERIES_INTRADAY_EXTENDED", symbol, "1min", yearMonth, apiKey,
				) // downloads 1 minute interval by default

				// download data (large un-streamed data, add streaming?)
				response, err := akitohttp.RequestGET(&httpClient, downloadURL)
				if err != nil { // handle err
					return // not configured to handle errors...
				}

				downloaded[index] = make([]byte, len(response.Body()))
				copy(downloaded[index], response.Body()) // copy(dest, src)
				index++                                  // increment for next month

				fasthttp.ReleaseResponse(response) // note: empties response.Body
			}(year, month)
		}
	}
	wg.Wait()

	fmt.Printf("Finished downloading %s CSV data\n", symbol)

	return downloaded
}

// parseIntradayExt parses the CSV data line-by-line into a struct for eventual marshalling.
// Returns the parsed ticker data and nil error if none encountered, otherwise returns nil pointer and the error.
// Times are segregated by year, month, week (only contains week data), day, and minute.
// Statistics like open/close, etc. are also configured here for the struct.
// * Note that pre-market and after-market price points are unreliable.
// TODO pricing and volume is not included if pre-market or after-market, make sure correct
func parseIntradayExt(csvData [][]byte, symbol string) (*tickerData, error) {
	// initialize container struct and contained maps
	ticker := tickerData{
		Symbol: symbol,
		Years:  make(map[int]*yearData),
	}

	// parse csv data into struct
	var err error
	var allRecords [][]string // save memory
	fmt.Printf("Parsing %s CSV data to struct ...\n", symbol)
	for _, data := range csvData { // iterate over downloaded files
		csvReader := csv.NewReader(bytes.NewReader(data)) // init csv reader
		_, _ = csvReader.Read()                           // read and ignore headers

		// time, open, high, low, close, volume
		allRecords, err = csvReader.ReadAll()
		if err != nil {
			return nil, err
		}

		var records []string                // save memory
		for _, records = range allRecords { // read while line exists
			var ok bool // for future use

			// parse time into individual values
			tim, _ := time.Parse(dateFormat, records[0])
			year, month, day, hour, minute := tim.Year(), int(tim.Month()), tim.Day(), tim.Hour(), tim.Minute() // integers

			// parse data to integers/floats as needed
			// assume that data is always in correct format
			open, _ := strconv.ParseFloat(records[1], 64)
			high, _ := strconv.ParseFloat(records[2], 64)
			low, _ := strconv.ParseFloat(records[3], 64)
			closeP, _ := strconv.ParseFloat(records[4], 64) // "close" collides with built-in
			volume, _ := strconv.Atoi(records[5])

			var dYear *yearData
			if dYear, ok = ticker.Years[year]; !ok {
				ticker.Years[year] = &yearData{Months: map[int]*monthData{}}
				dYear = ticker.Years[year]
			} // check year initialization

			var dMonth *monthData
			if dMonth, ok = dYear.Months[month]; !ok {
				dYear.Months[month] = &monthData{Days: map[int]*dayData{}}
				dMonth = dYear.Months[month]
			} // check month initialization

			var dDay *dayData
			if dDay, ok = dMonth.Days[day]; !ok {
				dMonth.Days[day] = &dayData{Points: map[dayTime]*pricing{}}
				dDay = dMonth.Days[day]
			} // check day initialization

			// TRUNCATE (floor) floats to 2 decimal places
			dDay.Points[dayTime{hour, minute}] = &pricing{
				Open:   math.Floor(open*100) / 100,
				Close:  math.Floor(closeP*100) / 100,
				High:   math.Floor(high*100) / 100,
				Low:    math.Floor(low*100) / 100,
				Volume: volume,
			} // initialize intraday point (per minute)
		}
	}

	// calculate statistical data (all by day?)
	// assume that all statistics are initialized as zero
	for year, dYear := range ticker.Years { // calculate year statistics
		for month, dMonth := range dYear.Months { // calculate month statistics
			fmt.Printf("Calculating statistics for %d/%d ...\n", month, year)

			// find first and last trading day
			var index int
			days := make([]int, len(dMonth.Days))
			for key := range dMonth.Days {
				days[index] = key
				index++
			}
			sort.Ints(days)

			dMonth.Open = dMonth.Days[days[0]].Open
			dMonth.Close = dMonth.Days[days[len(days)-1]].Close

			for _, dDay := range dMonth.Days { // calculate day statistics

				// market sometimes closes early or something, get last trading point
				var times []dayTime
				for key := range dDay.Points {
					// exclude after-hours trading closing time
					if (key.Hour > 16) || (key.Hour == 16 && key.Minute > 0) {
						continue
					}

					times = append(times, key)
				}
				sort.Sort(sDayTime(times))

				dDay.Open = dDay.Points[openTime].Open
				dDay.Close = dDay.Points[times[len(times)-1]].Close

				dDay.High = 0                           // init high
				dDay.Low = math.MaxFloat64              // init low
				for dTime, point := range dDay.Points { // iterate over price points
					// exclude premarket (9:30 inclusive) and aftermarket (16:00 exclusive)
					if !((dTime.Hour <= 9 && dTime.Minute <= 30) || (dTime.Hour >= 16 && dTime.Minute > 0)) {
						dDay.Volume += point.Volume // add volume for later division

						// process high and low prices
						if point.High > dDay.High {
							dDay.High = point.High
						}
						if point.High < dDay.High {
							dDay.High = point.High
						}
					}
				}
			}
		}
	}

	return &ticker, nil
}
