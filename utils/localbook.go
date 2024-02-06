package utils

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/adshao/go-binance/v2"
)

var (
	peaky      *log.Logger
	RUNTIME    = (2 * time.Minute)
	TICKERTIME = (15 * time.Second)
)

func init() {
	peaky = log.New(os.Stdout, "peaky:", log.LstdFlags)
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
