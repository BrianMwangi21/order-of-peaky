package main

import (
	"sync"

	"github.com/BrianMwangi21/order-of-peaky.git/localbook"
)

var (
	SYMBOLS = [...]string{"BTCUSDT", "ETHUSDT"}
)

func main() {
	var wg sync.WaitGroup

	for _, symbol := range SYMBOLS {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			localbook.Begin(s)
		}(symbol)
	}

	wg.Wait()
}
