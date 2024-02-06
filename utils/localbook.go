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
	peaky                 *log.Logger
	ORDER_BOOK_SIZE_LIMIT = 50
	SYMBOL                string
)

func init() {
	peaky = log.New(os.Stdout, "peaky:", log.LstdFlags|log.Lshortfile)
	peaky.Println("Logger initialized")
}

func getDepthSnapshot() DepthSnapshot {
	var snapshot DepthSnapshot
	resp, err := http.Get("https://api.binance.com/api/v3/depth?symbol=" + SYMBOL + "&limit=5000")
	if err != nil {
		peaky.Fatalf("%s : Error fetching snapshot from API: %v", SYMBOL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		peaky.Fatalf("%s : Error reading response body: %v", SYMBOL, err)
	}

	err = json.Unmarshal(body, &snapshot)
	if err != nil {
		peaky.Fatalf("%s : Error unmarshalling depth snapshot: %v", SYMBOL, err)
	}

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
	} else {
		stopC <- struct{}{}
	}
}

func wsErrorHandler(err error) {
	if err != nil {
		peaky.Fatalf("%s : WsDepthServe error: %v", SYMBOL, err)
	}
}

func ManageLocalOrderBook(symbol string) {
	SYMBOL = symbol
	snapshot := getDepthSnapshot()
	orderBookLock.Lock()
	lastUpdateId = snapshot.LastUpdateId
	orderBookLock.Unlock()

	peaky.Printf("%s : Starting events buffering", SYMBOL)
	peaky.Printf("%s : Depth Snapshot lastUpdateId: %d\n", SYMBOL, lastUpdateId)

	doneC, stopC, wsErr = binance.WsDepthServe(symbol, wsDepthHander, wsErrorHandler)
	if wsErr != nil {
		peaky.Fatalf("%s : Error launching WsDepthServe websocket: %v\n", SYMBOL, wsErr)
	}

	<-doneC

	peaky.Printf("%s : Finished events buffering. Events stored: %d\n", SYMBOL, len(orderBook))
}
