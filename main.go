package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const apiKey = "6XOID1CXG2FACMV8"

func main() {
	tickerFile, err := os.Open("companylist.csv")
	if err != nil {
		panic(err)
	}
	allTickerData, _ := csv.NewReader(tickerFile).ReadAll()

	var index int
	for _, tickerData := range allTickerData {
		if tickerData[5] != "Technology" && !strings.Contains(tickerData[3], "B") {
			continue
		}

		index++

		fmt.Printf("[%d] Downloading and parsing intraday extended trading data for %s ...\n", index, tickerData[0])

		csvData := downloadIntradayExt(apiKey, tickerData[0])
		ticker, err := parseIntradayExt(csvData, tickerData[0])
		if err != nil {
			panic(err)
		}

		file, err := os.Create("technology/" + tickerData[0] + ".json")
		if err != nil {
			panic(err)
		}
		marshalled, err := json.Marshal(ticker) // bc pointer
		if err != nil {
			panic(err)
		}
		_, _ = file.Write(marshalled)
		_ = file.Close()
	}
}
