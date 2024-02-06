package localbook

import (
	"fmt"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2"
)

var (
	peaky      *log.Logger
	RUNTIME    = (15 * time.Minute)
	TICKERTIME = (30 * time.Second)
)

func init() {
	peaky = log.New(os.Stdout, "peaky:", log.LstdFlags)
}

type OrderBook struct {
	sync.RWMutex
	Bids          map[float64]float64
	Asks          map[float64]float64
	Symbol        string
	LastUpdateId  int64
	doneC, stopC  chan struct{}
	wsErr         error
	eventsCounter int
}

func newOrderBook(symbol string, lastUpdateId int64) *OrderBook {
	return &OrderBook{
		Bids:         make(map[float64]float64),
		Asks:         make(map[float64]float64),
		Symbol:       symbol,
		LastUpdateId: lastUpdateId,
	}
}

func (ob *OrderBook) wsErrorHandler(err error) {
	if err != nil {
		peaky.Fatalf("%s : WsDepthServe error: %v", ob.Symbol, err)
	}
}

func (ob *OrderBook) wsDepthHander(event *binance.WsDepthEvent) {
	if event.LastUpdateID <= ob.LastUpdateId {
		return
	}

	ob.eventsCounter += 1
	peaky.Printf("%s : Processing Event: %d, Bids: %d, Asks: %d\n", ob.Symbol, event.LastUpdateID, len(event.Bids), len(event.Asks))
	ob.updateOrderBook(event)
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

	totalBids, totalAsks := ob.getTotalBidsAsks()
	lowestAsk, highestBid, spread := ob.getSpread()
	peaky.Printf("%s : Total Bids: %.4f, Total Asks: %.4f, Lowest Ask: %.4f, Highest Bid: %.4f, Spread: %.4f\n", ob.Symbol, totalBids, totalAsks, lowestAsk, highestBid, spread)
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

func Begin(symbol string) {
	snapshot := getDepthSnapshot(symbol)
	ob := newOrderBook(symbol, snapshot.LastUpdateId)

	peaky.Printf("%s : Depth Snapshot lastUpdateId: %d\n", ob.Symbol, ob.LastUpdateId)
	peaky.Printf("%s : Starting events capturing\n", ob.Symbol)

	ob.doneC, ob.stopC, ob.wsErr = binance.WsDepthServe(symbol, ob.wsDepthHander, ob.wsErrorHandler)
	if ob.wsErr != nil {
		peaky.Fatalf("%s : Error launching WsDepthServe websocket: %v\n", ob.Symbol, ob.wsErr)
	}

	ticker := time.NewTicker(TICKERTIME)

	go func(ob *OrderBook) {
		time.Sleep(RUNTIME)
		ob.stopC <- struct{}{}
	}(ob)

	go func(ob *OrderBook) {
		for {
			select {
			case <-ticker.C:
				ob.displaySentiments()
			case <-ob.stopC:
				ticker.Stop()
				return
			}
		}
	}(ob)

	<-ob.doneC

	fmt.Fprintln(os.Stdout)
	peaky.Printf("%s : Finished events capturing. Events processed: %d\n", ob.Symbol, ob.eventsCounter)
}
