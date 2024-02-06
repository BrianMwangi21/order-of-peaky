package localbook

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/adshao/go-binance/v2/common"
)

type DepthSnapshot struct {
	LastUpdateId int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

func parsePriceLevel(pl []common.PriceLevel) (totalQuantity float64) {
	for _, level := range pl {
		totalQuantity += parseToFloat(level.Quantity)
	}
	return totalQuantity
}

func parseToFloat(input string) (value float64) {
	value, err := strconv.ParseFloat(input, 64)
	if err != nil {
		peaky.Fatalf("Error parsing to float: %v", err)
	}
	return value
}

func getDepthSnapshot(symbol string) DepthSnapshot {
	var snapshot DepthSnapshot
	resp, err := http.Get("https://api.binance.com/api/v3/depth?symbol=" + symbol + "&limit=5000")
	if err != nil {
		peaky.Fatalf("%s : Error fetching snapshot from API: %v", symbol, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		peaky.Fatalf("%s : Error reading response body: %v", symbol, err)
	}

	err = json.Unmarshal(body, &snapshot)
	if err != nil {
		peaky.Fatalf("%s : Error unmarshalling depth snapshot: %v", symbol, err)
	}

	return snapshot
}
