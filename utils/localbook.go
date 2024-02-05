package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/adshao/go-binance/v2"
)

type DepthSnapshot struct {
	LastUpdateId int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

var (
	lastUpdateId          int64
	orderBookLock         sync.Mutex
	orderBook             []*binance.WsDepthEvent
	doneC, stopC          chan struct{}
	wsErr                 error
	ORDER_BOOK_SIZE_LIMIT = 100
)

func getDepthSnapshot(symbol string) DepthSnapshot {
	var snapshot DepthSnapshot
	resp, err := http.Get("https://api.binance.com/api/v3/depth?symbol=" + symbol + "&limit=5000")
	HandleError("Error fetching snapshot from API:", err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	HandleError("Error reading response body:", err)

	err = json.Unmarshal(body, &snapshot)
	HandleError("Error unmarshalling depth snapshot:", err)

	return snapshot
}

func wsDepthHander(event *binance.WsDepthEvent) {
	orderBookLock.Lock()
	defer orderBookLock.Unlock()

	if event.LastUpdateID <= lastUpdateId {
		return
	}

	if len(orderBook) < ORDER_BOOK_SIZE_LIMIT {
		orderBook = append(orderBook, event)
		fmt.Println("Event received and processed:", event.LastUpdateID, len(event.Bids), len(event.Asks))
	} else {
		stopC <- struct{}{}
	}
}

func wsErrorHandler(err error) {
	HandleError("WsDepthServe error:", err)
}

func ManageLocalOrderBook(symbol string) {
	snapshot := getDepthSnapshot(symbol)
	orderBookLock.Lock()
	lastUpdateId = snapshot.LastUpdateId
	orderBookLock.Unlock()

	fmt.Println("Depth Snapshot lastUpdateId:", lastUpdateId)

	doneC, stopC, wsErr = binance.WsDepthServe(symbol, wsDepthHander, wsErrorHandler)
	HandleError("Error launching WsDepthServe websocket:", wsErr)

	<-doneC

	fmt.Println("Orderbook len", len(orderBook))
}
