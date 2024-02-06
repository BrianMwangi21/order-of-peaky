package main

import (
	"sync"

	"github.com/BrianMwangi21/order-of-peaky.git/utils"
)

var (
	SYMBOLS = [...]string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
)

func main() {
	var wg sync.WaitGroup

	for _, symbol := range SYMBOLS {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			utils.Begin(s)
		}(symbol)
	}

	wg.Wait()
}
