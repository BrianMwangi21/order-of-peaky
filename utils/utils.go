package utils

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/common"
)

type DepthSnapshot struct {
	LastUpdateId int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

type OrderBook struct {
	Bids map[float64]float64
	Asks map[float64]float64
}

func newOrderBook() *OrderBook {
	return &OrderBook{
		Bids: make(map[float64]float64),
		Asks: make(map[float64]float64),
	}
}

func (ob *OrderBook) updateOrderBook(event *binance.WsDepthEvent) {
	totalBidQuantity += parsePriceLevel(event.Bids)
	totalAskQuantity += parsePriceLevel(event.Asks)

	// Update Bids
	for _, bid := range event.Bids {
		price := parseToFloat(bid.Price)
		quantity := parseToFloat(bid.Quantity)

		if quantity == 0 {
			delete(ob.Bids, price)
		} else {
			ob.Bids[price] = quantity
		}
	}

	// Update Asks
	for _, ask := range event.Asks {
		price := parseToFloat(ask.Price)
		quantity := parseToFloat(ask.Quantity)

		if quantity == 0 {
			delete(ob.Asks, price)
		} else {
			ob.Asks[price] = quantity
		}
	}
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
		peaky.Fatalf("%s : Error parsing to float: %v", SYMBOL, err)
	}
	return value
}

func getDepthSnapshot() DepthSnapshot {
	var snapshot DepthSnapshot
	resp, err := http.Get("https://api.binance.com/api/v3/depth?symbol=" + SYMBOL + "&limit=5000")
	if err != nil {
		peaky.Fatalf("%s : Error fetching snapshot from API: %v", SYMBOL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		peaky.Fatalf("%s : Error reading response body: %v", SYMBOL, err)
	}

	err = json.Unmarshal(body, &snapshot)
	if err != nil {
		peaky.Fatalf("%s : Error unmarshalling depth snapshot: %v", SYMBOL, err)
	}

	return snapshot
}
