package main

import (
	"encoding/json"
	"fmt"
)

/*
Automated downloading of stocks pricing history from AlphaVantage via their TIME_SERIES_INTRADAY_EXTENDED function.
Concurrently downloads 24 "slices" of data from year1month1 to year2month12 (option for concurrent?)
Crops trading data from 9:30AM to 4:00PM, thus excluding pre-market and after-hours trading data.
Stores ticker data within a single JSON file separated into days (with fields for high, low, etc.)

[year [month [week] [day]]] data structure where [] represents the contents of a map.
*/

// tickerData contains collected data for a single stock ticker/symbol.
// Data is separated into individual data structures for year, month, week, and day.
type tickerData struct {
	Symbol string // for reference
	// maybe add category, name, etc. for reference
	Years map[int]*yearData // per year
}

// pricing contains basic pricing information.
type pricing struct {
	Open   float64 // opening price
	Close  float64 // closing price
	High   float64
	Low    float64
	Volume int
}

// yearData contains year-long statistical data and month structures.
type yearData struct {
	Months map[int]*monthData // per month
}

// monthData contains month-long statistical data and day/month structures.
type monthData struct {
	pricing
	Days map[int]*dayData // per day
}

// dayData contains minute-long interval points for pricing, etc.
type dayData struct {
	pricing
	Points map[dayTime]*pricing // better way?
}

// dayTime contains hour and minute data for easy retrieval of times.
type dayTime struct {
	Hour   int // 24-hour format, important
	Minute int // between 0 and 59
}

// custom marshal keys (HH:MM)
func (d dayTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%2d:%2d", d.Hour, d.Minute))
}
func (d dayTime) MarshalText() (text []byte, err error) {
	return []byte(fmt.Sprintf("%2d:%2d", d.Hour, d.Minute)), nil
}

type sDayTime []dayTime // for sorting purposes
func (d sDayTime) Len() int {
	return len(d)
}
func (d sDayTime) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
func (d sDayTime) Less(i, j int) bool {
	if d[i].Hour < d[j].Hour {
		return true
	} else if d[i].Hour == d[j].Hour {
		if d[i].Minute < d[j].Minute {
			return true
		}

		return false
	}

	return false
}
