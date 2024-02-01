package main

import (
	"fmt"
	"log"
	"time"

	"github.com/BrianMwangi21/order-of-peaky.git/configs"
	"github.com/adshao/go-binance/v2"
)

var (
	BINANCE_API_KEY    string
	BINANCE_SECRET_KEY string
)

func init() {
	BINANCE_API_KEY, BINANCE_SECRET_KEY = configs.GetBinanceKeys()
}

func wsDepthHandler(event *binance.WsDepthEvent) {
	fmt.Println("----Event received----")
	fmt.Println("Event:", event.Event)
	fmt.Println("Time:", event.Time)
	fmt.Println("Symbol:", event.Symbol)
	fmt.Println("LastUpdateID:", event.LastUpdateID)
	fmt.Println("FirstUpdateID:", event.FirstUpdateID)
	fmt.Println("Bids count:", len(event.Bids))
	fmt.Println("Asks count:", len(event.Asks))
	fmt.Println("------Event  end------")
}

func errHandler(err error) {
	log.Fatal(err)
}

func main() {
	doneC, stopC, err := binance.WsDepthServe("BTCUSDT", wsDepthHandler, errHandler)
	if err != nil {
		fmt.Println(err)
		return
	}
	// use stopC to exit
	go func() {
		time.Sleep(50 * time.Second)
		stopC <- struct{}{}
	}()
	// remove this if you do not want to be blocked here
	<-doneC
}
