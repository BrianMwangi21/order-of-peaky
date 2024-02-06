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

type OrderBook struct {
	sync.RWMutex
	Bids                      map[float64]float64
	Asks                      map[float64]float64
	Symbol                    string
	SnapshotLastUpdateId      int64
	doneC, stopC              chan struct{}
	wsErr                     error
	eventsCounter             int
	previousEventLastUpdateId int64
}

func init() {
	peaky = log.New(os.Stdout, "peaky:", log.LstdFlags)
}

func newOrderBook(symbol string, lastUpdateId int64) *OrderBook {
	return &OrderBook{
		Bids:                 make(map[float64]float64),
		Asks:                 make(map[float64]float64),
		Symbol:               symbol,
		SnapshotLastUpdateId: lastUpdateId,
	}
}

func (ob *OrderBook) wsErrorHandler(err error) {
	if err != nil {
		peaky.Fatalf("%s : WsDepthServe error: %v", ob.Symbol, err)
	}
}

func (ob *OrderBook) wsDepthHander(event *binance.WsDepthEvent) {
	if !ob.isValidEvent(event) {
		return
	}

	ob.Lock()
	defer ob.Unlock()

	ob.eventsCounter += 1
	ob.previousEventLastUpdateId = event.LastUpdateID

	peaky.Printf("%s : Processing Event: F: %d, L: %d, Bids: %d, Asks: %d\n", ob.Symbol, event.FirstUpdateID, event.LastUpdateID, len(event.Bids), len(event.Asks))
	ob.updateOrderBook(event)
}

func (ob *OrderBook) isValidEvent(event *binance.WsDepthEvent) bool {
	// Guide : https://binance-docs.github.io/apidocs/delivery/en/#how-to-manage-a-local-order-book-correctly
	// Don't process any event where u <= SnapshotLastUpdateId
	if event.LastUpdateID <= ob.SnapshotLastUpdateId {
		peaky.Printf("%s : Event not valid. Check u <= SnapshotLastUpdateId failed: %d, %d\n", ob.Symbol, event.LastUpdateID, ob.SnapshotLastUpdateId)
		return false
	}

	// Event should also be such that U <= SnapshotLastUpdateId+1 and u >= SnapshotLastUpdateId+1
	if !(event.FirstUpdateID <= ob.SnapshotLastUpdateId+1) && !(event.LastUpdateID >= ob.SnapshotLastUpdateId+1) {
		peaky.Printf("%s : Event not valid. Check U <= SnapshotLastUpdateId+1 and u >= SnapshotLastUpdateId+1 failed: %d, %d, %d\n", ob.Symbol, event.FirstUpdateID, event.LastUpdateID, ob.SnapshotLastUpdateId+1)
		return false
	}

	// Event's U should be equal to previous event's u+1
	if event.FirstUpdateID != (ob.previousEventLastUpdateId + 1) {
		ob.Lock()
		ob.previousEventLastUpdateId = event.LastUpdateID
		ob.Unlock()
		return false
	}

	return true
}

func (ob *OrderBook) updateOrderBook(event *binance.WsDepthEvent) {
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

	peaky.Printf("%s : Depth Snapshot lastUpdateId: %d\n", ob.Symbol, ob.SnapshotLastUpdateId)
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
