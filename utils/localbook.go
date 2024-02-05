package utils

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
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
	peaky                 *log.Logger
)

func init() {
	peaky = log.New(os.Stdout, "peaky:", log.LstdFlags|log.Lshortfile)
	peaky.Println("Logger initialized")
}

func getDepthSnapshot(symbol string) DepthSnapshot {
	var snapshot DepthSnapshot
	resp, err := http.Get("https://api.binance.com/api/v3/depth?symbol=" + symbol + "&limit=5000")
	if err != nil {
		peaky.Fatalf("Error fetching snapshot from API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		peaky.Fatalf("Error reading response body: %v", err)
	}

	err = json.Unmarshal(body, &snapshot)
	if err != nil {
		peaky.Fatalf("Error unmarshalling depth snapshot: %v", err)
	}

	return snapshot
}

func wsDepthHander(event *binance.WsDepthEvent) {
	orderBookLock.Lock()
	defer orderBookLock.Unlock()

	if event.LastUpdateID <= lastUpdateId {
		peaky.Println("Event received and ignored:", event.LastUpdateID, len(event.Bids), len(event.Asks))
		return
	}

	if len(orderBook) < ORDER_BOOK_SIZE_LIMIT {
		orderBook = append(orderBook, event)
		peaky.Println("Event received and processed:", event.LastUpdateID, len(event.Bids), len(event.Asks))
	} else {
		stopC <- struct{}{}
	}
}

func wsErrorHandler(err error) {
	if err != nil {
		peaky.Fatalf("WsDepthServe error: %v", err)
	}
}

func ManageLocalOrderBook(symbol string) {
	snapshot := getDepthSnapshot(symbol)
	orderBookLock.Lock()
	lastUpdateId = snapshot.LastUpdateId
	orderBookLock.Unlock()

	peaky.Println("Depth Snapshot lastUpdateId:", lastUpdateId)

	doneC, stopC, wsErr = binance.WsDepthServe(symbol, wsDepthHander, wsErrorHandler)
	if wsErr != nil {
		peaky.Fatalf("Error launching WsDepthServe websocket: %v", wsErr)
	}

	<-doneC

	peaky.Println("Orderbook length", len(orderBook))
}
