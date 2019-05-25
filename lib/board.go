package bitflyerspider

import (
    "fmt"
    "github.com/mitsutoshi/bitflyergo"
    "github.com/mitsutoshi/bitflyerspider/lib/bq"
    "math"
    "sort"
    "sync"
    "time"
)

type BoardCollector struct {
    Asks          map[float64]float64
    Bids          map[float64]float64
    AskPrices     []float64
    BidPrices     []float64
    BoardMutex    *sync.Mutex
    SummaryPerSec []*bq.BqBoard
    AggInterval   time.Duration
}

func NewCollector() *BoardCollector {
    return &BoardCollector{
        Asks:        map[float64]float64{},
        Bids:        map[float64]float64{},
        BoardMutex:  &sync.Mutex{},
        AggInterval: 1 * time.Second,
    }
}

func (c *BoardCollector) BestAskPrice() float64 {
    if len(c.AskPrices) > 0 {
        return c.AskPrices[0]
    }
    return 0
}

func (c *BoardCollector) BestBidPrice() float64 {
    if len(c.BidPrices) > 0 {
        return c.BidPrices[0]
    }
    return 0
}

func (c *BoardCollector) Spread() float64 {
    return c.BestAskPrice() - c.BestBidPrice()
}

func (c *BoardCollector) MidPrice() float64 {
    return c.BestBidPrice() + math.Round(c.Spread()/2)
}

// Update board
func (c *BoardCollector) UpdateBoard(newBoard *bitflyergo.Board, refresh bool) {

    c.BoardMutex.Lock()
    defer c.BoardMutex.Unlock()

    // refreshが指定された場合は既存の板を破棄して作り直す
    if refresh {
        c.Asks = map[float64]float64{}
        c.Bids = map[float64]float64{}
    }

    // Update Asks
    for price, size := range newBoard.Asks {
        if size > 0 {

            // Add or Update
            c.Asks[price] = size
        } else if _, ok := c.Asks[price]; ok {

            // Delete
            delete(c.Asks, price)
        }
    }

    // Update Bids
    for price, size := range newBoard.Bids {
        if size > 0 {

            // Add or Update
            c.Bids[price] = size
        } else if _, ok := c.Bids[price]; ok {

            // Delete
            delete(c.Bids, price)
        }
    }

    // Update best ask, best bid, mid price and spread.
    if len(c.Asks) > 0 {
        c.sortAsks()
    }
    if len(c.Bids) > 0 {
        c.sortBids()
    }
}

func (c *BoardCollector) sortAsks() {
    c.AskPrices = make([]float64, 0, len(c.Asks))
    for k := range c.Asks {
        c.AskPrices = append(c.AskPrices, k)
    }
    sort.Sort(sort.Float64Slice(c.AskPrices))
}

func (c *BoardCollector) sortBids() {
    c.BidPrices = make([]float64, 0, len(c.Bids))
    for k := range c.Bids {
        c.BidPrices = append(c.BidPrices, k)
    }
    sort.Sort(sort.Reverse(sort.Float64Slice(c.BidPrices)))
}

func (c *BoardCollector) Agg() {

    const interval = 200 * time.Millisecond
    nextTime := time.Now()

    for {
        time.Sleep(interval)

        // 集計時刻を過ぎたら集計して出力
        if time.Now().After(nextTime) {

            // BigQueryレコードの型に変換
            c.BoardMutex.Lock()
            askPrice := c.BestAskPrice()
            bidPrice := c.BestBidPrice()
            if askPrice > 0 && bidPrice > 0 {
                c.SummaryPerSec = append(c.SummaryPerSec, &bq.BqBoard{
                    Time:         nextTime,
                    MidPrice:     int(c.MidPrice()),
                    BestAskPrice: int(askPrice),
                    BestBidPrice: int(bidPrice),
                    BestAskSize:  c.Asks[askPrice],
                    BestBidSize:  c.Bids[bidPrice],
                    Spread:       int(c.Spread()),
                })
            }
            c.BoardMutex.Unlock()

            fmt.Printf("%v -> mid: %v, ask: %v, bid: %v, spread: %v\n",
                nextTime.Format("2006-01-02 15:04:05"), c.MidPrice(), askPrice, bidPrice, c.Spread())

            // 次の集計時刻を計算
            nextTime = nextTime.Add(c.AggInterval)
        }
    }
}
