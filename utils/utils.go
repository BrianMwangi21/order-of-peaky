package utils

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/common"
)

type DepthSnapshot struct {
	LastUpdateId int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

type OrderBook struct {
	sync.RWMutex
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
	ob.Lock()
	defer ob.Unlock()

	for _, bid := range event.Bids {
		price := parseToFloat(bid.Price)
		quantity := parseToFloat(bid.Quantity)

		if quantity == 0 {
			delete(ob.Bids, price)
		} else {
			ob.Bids[price] = quantity
		}
	}

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

func (ob *OrderBook) displaySentiments() {
	ob.RLock()
	defer ob.RUnlock()

	totalBids, totalAsks := orderBook.getTotalBidsAsks()
	lowestAsk, highestBid, spread := orderBook.getSpread()
	peaky.Printf("%s : Total Bids: %.4f, Total Asks: %.4f, Lowest Ask: %.4f, Highest Bid: %.4f, Spread: %.4f\n", SYMBOL, totalBids, totalAsks, lowestAsk, highestBid, spread)
}

func (ob *OrderBook) getTotalBidsAsks() (totalBids, totalAsks float64) {
	for _, quantity := range ob.Bids {
		totalBids += quantity
	}
	for _, quantity := range ob.Asks {
		totalAsks += quantity
	}

	return totalBids, totalAsks
}

func (ob *OrderBook) getSpread() (lowestAsk, highestBid, spread float64) {
	if len(ob.Asks) == 0 || len(ob.Bids) == 0 {
		return math.NaN(), math.NaN(), math.NaN()
	}

	lowestAsk = math.Inf(1)
	highestBid = math.Inf(-1)

	for price := range ob.Asks {
		if price < lowestAsk {
			lowestAsk = price
		}
	}
	for price := range ob.Bids {
		if price > highestBid {
			highestBid = price
		}
	}

	spread = lowestAsk - highestBid
	return lowestAsk, highestBid, spread
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
