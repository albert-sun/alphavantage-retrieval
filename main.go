package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/remeh/sizedwaitgroup"
)

const apiKey = "6XOID1CXG2FACMV8"

func main() {
	tickerFile, err := os.Open("companylist.csv")
	if err != nil {
		panic(err)
	}
	allTickerData, _ := csv.NewReader(tickerFile).ReadAll()

	var index int
	swg := sizedwaitgroup.New(5) // too many requests makes the server scream?
	for _, tickerData := range allTickerData {
		tickerData[0] = strings.TrimSpace(tickerData[0]) // trim leading and trailing whitespace
		if tickerData[5] != "Technology" && !strings.Contains(tickerData[3], "B") {
			continue
		}

		swg.Add()
		go func(symbol string) {
			defer swg.Done()

			csvData := downloadIntradayExt(apiKey, symbol)
			ticker, err := parseIntradayExt(csvData, symbol)
			if err != nil {
				panic(err)
			}

			file, err := os.Create("technology/" + symbol + ".json")
			if err != nil {
				panic(err)
			}
			marshalled, err := json.Marshal(ticker) // bc pointer
			if err != nil {
				panic(err)
			}
			_, _ = file.Write(marshalled)
			_ = file.Close()

			index++
			fmt.Printf("[%d] Finished downloading and parsing intraday extended trading data for %s ...\n", index, symbol)
		}(tickerData[0])
	}

	swg.Wait()
}
