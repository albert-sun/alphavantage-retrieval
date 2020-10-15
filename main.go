package main

import (
	"encoding/csv"
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
	allTickerData = allTickerData[1 : len(allTickerData)-1] // ignore header

	var index int
	swg := sizedwaitgroup.New(5) // too many requests makes the server scream?
	for _, tickerData := range allTickerData {
		tickerData[0] = strings.TrimSpace(tickerData[0]) // trim leading and trailing whitespace

		swg.Add()
		go func(symbol string) {
			defer swg.Done()

			csvData := downloadIntradayExt(apiKey, symbol)
			for ind, csv := range csvData {
				filename := fmt.Sprintf("year%dmonth%d.csv", ind/12+1, ind%12+1)
				_ = os.Mkdir(fmt.Sprintf("intraday-extended-csv/%s", symbol), 0755)
				file, _ := os.Create(fmt.Sprintf("intraday-extended-csv/%s/%s", symbol, filename))

				_, _ = file.Write(csv)
				_ = file.Close()
			}

			index++
			fmt.Printf("[%d] Finished caching CSV data for %s ...\n", index, symbol)
		}(tickerData[0])
	}

	swg.Wait()
}
