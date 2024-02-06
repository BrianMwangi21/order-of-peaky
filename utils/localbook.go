package utils

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/adshao/go-binance/v2"
)

var (
	lastUpdateId                       int64
	doneC, stopC                       chan struct{}
	wsErr                              error
	peaky                              *log.Logger
	totalBidQuantity, totalAskQuantity float64
	eventsCounter                      int
	orderBook                          *OrderBook
	SYMBOL                             string
	RUNTIME                            = (2 * time.Minute)
)

func init() {
	orderBook = newOrderBook()
	peaky = log.New(os.Stdout, "peaky:", log.LstdFlags|log.Lshortfile)
	peaky.Println("Logger initialized")
}

func wsDepthHander(event *binance.WsDepthEvent) {
	if event.LastUpdateID <= lastUpdateId {
		return
	}

	eventsCounter += 1
	peaky.Printf("%s : Processing Event: %d, Bids: %d, Asks: %d\n", SYMBOL, event.LastUpdateID, len(event.Bids), len(event.Asks))
	orderBook.updateOrderBook(event)
}

func wsErrorHandler(err error) {
	if err != nil {
		peaky.Fatalf("%s : WsDepthServe error: %v", SYMBOL, err)
	}
}

func Begin(symbol string) {
	SYMBOL = symbol
	snapshot := getDepthSnapshot()
	lastUpdateId = snapshot.LastUpdateId

	peaky.Printf("%s : Depth Snapshot lastUpdateId: %d\n", SYMBOL, lastUpdateId)
	peaky.Printf("%s : Starting events capturing\n", SYMBOL)

	doneC, stopC, wsErr = binance.WsDepthServe(symbol, wsDepthHander, wsErrorHandler)
	if wsErr != nil {
		peaky.Fatalf("%s : Error launching WsDepthServe websocket: %v\n", SYMBOL, wsErr)
	}
	go func() {
		time.Sleep(RUNTIME)
		stopC <- struct{}{}
	}()

	<-doneC

	fmt.Fprintln(os.Stdout)
	peaky.Printf("%s : Finished events capturing. Events processed: %d\n", SYMBOL, eventsCounter)
}
