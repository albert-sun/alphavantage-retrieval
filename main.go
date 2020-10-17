package main

import (
	"encoding/csv"
	"fmt"
	"github.com/remeh/sizedwaitgroup"
	"os"
	"strings"
)

// maps folder name to URL for retrieval (all JSON format)
var functionMap = map[string]string{
	"intraday":         "https://www.alphavantage.co/query?function=TIME_SERIES_INTRADAY&symbol=%s&interval=1min&outputsize=full&apikey=%s",
	"daily":            "https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&outputsize=full&apikey=%s",
	"daily-adjusted":   "https://www.alphavantage.co/query?function=TIME_SERIES_DAILY_ADJUSTED&symbol=%s&outputsize=full&apikey=%s",
	"weekly":           "https://www.alphavantage.co/query?function=TIME_SERIES_WEEKLY&symbol=%s&outputsize=full&apikey=%s",
	"weekly-adjusted":  "https://www.alphavantage.co/query?function=TIME_SERIES_WEEKLY_ADJUSTED&symbol=%s&outputsize=full&apikey=%s",
	"monthly":          "https://www.alphavantage.co/query?function=TIME_SERIES_MONTHLY&symbol=%s&outputsize=full&apikey=%s",
	"monthly-adjusted": "https://www.alphavantage.co/query?function=TIME_SERIES_MONTHLY_ADJUSTED&symbol=%s&outputsize=full&apikey=%s",
	"quote":            "https://www.alphavantage.co/query?function=TIME_SERIES_GLOBAL_QUOTE&symbol=%s&apikey=%s",
}

func main() {
	fmt.Println("Initializing client before caching ...")
	client := newAlphaClient("KMYXJ842XBY3XWKQ", 25) // initialize client, 25<30

	// read and parse ticker data from file
	tickerFile, _ := os.Open("companylist.csv") // lazy
	allTickerData, _ := csv.NewReader(tickerFile).ReadAll()
	allTickerData = allTickerData[1 : len(allTickerData)-1] // "crop" headers

	// create folders if doesn't exist already
	for folderName := range functionMap {
		_ = os.Mkdir(folderName, 0755)
	}

	var index int // for tracking progress
	swg := sizedwaitgroup.New(5)
	for _, tickerData := range allTickerData {
		swg.Add()
		go func(symbol string) { // download all for one ticker
			defer swg.Done()

			for folderName, downloadURL := range functionMap {
				// check whether file already exists, skip if so
				if _, err := os.Open(fmt.Sprintf("%s/%s.json", folderName, symbol)); err == nil {
					continue
				}

				downloadData, err := client.download(fmt.Sprintf(downloadURL, symbol, client.apiKey))
				if err != nil {
					fmt.Printf("Error downloading [%s] data for %s\n", folderName, symbol)
				} else { // save data as json
					if strings.Contains(string(downloadData), `"Note"`) { // check rate limit
						fmt.Println("!! Warning: rate limiting in effect")
					}

					file, err := os.Create(fmt.Sprintf("%s/%s.json", folderName, symbol))
					if err != nil { // system-restricted name
						file, _ = os.Create(fmt.Sprintf("%s/%s_.json", folderName, symbol))
					}
					_, _ = file.Write(downloadData)
					_ = file.Close()
				}
			}

			index++
			fmt.Printf("[%d] Finished caching non-intraday-extended time data for %s\n", index, symbol)
		}(strings.TrimSpace(tickerData[0]))
	}
	swg.Wait()

	fmt.Printf("Finished downloading a total of %d stocks.\n", index)
}
