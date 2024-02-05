package main

import (
	"github.com/BrianMwangi21/order-of-peaky.git/utils"
)

var (
	SYMBOL = "BTCUSDT"
)

func main() {
	utils.ManageLocalOrderBook(SYMBOL)
}
