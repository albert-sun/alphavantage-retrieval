package main

import (
	"encoding/json"
	"os"
)

const apiKey = "6XOID1CXG2FACMV8"

func main() {
	csvData := downloadIntradayExt(apiKey, "MSFT")

	ticker, err := parseIntradayExt(csvData, "MSFT")
	if err != nil {
		panic(err)
	}

	file, err := os.Create("MSFT.json")
	if err != nil {
		panic(err)
	}
	marshalled, err := json.Marshal(ticker) // bc pointer
	if err != nil {
		panic(err)
	}

	_, _ = file.Write(marshalled)
}
