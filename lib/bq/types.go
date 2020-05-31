package bq

import (
	"cloud.google.com/go/bigquery"
	"time"
)

type BqExecution struct {
	Id                         int64     // ID
	ExecDate                   time.Time // 日時
	Price                      int       // 価格
	Size                       float64   // サイズ
	Side                       string    // 売買種別
	BuyChildOrderAcceptanceId  string    // 買い注文ID
	SellChildOrderAcceptanceId string    // 売り注文ID
	Delay                      float64   // 受信遅延時間
	ReceivedTime               time.Time
}

func (e *BqExecution) Save() (row map[string]bigquery.Value, insertID string, err error) {
	return map[string]bigquery.Value{
		"id":                             e.Id,
		"exec_date":                      e.ExecDate,
		"size":                           e.Size,
		"side":                           e.Side,
		"price":                          e.Price,
		"buy_child_order_acceptance_id":  e.BuyChildOrderAcceptanceId,
		"sell_child_order_acceptance_id": e.SellChildOrderAcceptanceId,
		"delay":                          e.Delay,
		"received_time":                  e.ReceivedTime,
	}, "", nil
}

type BqBoard struct {
	Time         time.Time
	MidPrice     int
	BestAskPrice int
	BestAskSize  float64
	BestBidPrice int
	BestBidSize  float64
	Spread       int
}

func (b *BqBoard) Save() (row map[string]bigquery.Value, insertID string, err error) {
	return map[string]bigquery.Value{
		"time":           b.Time,
		"mid_price":      b.MidPrice,
		"best_ask_price": b.BestAskPrice,
		"best_ask_size":  b.BestAskSize,
		"best_bid_price": b.BestBidPrice,
		"best_bid_size":  b.BestBidSize,
		"spread":         b.Spread,
	}, "", nil
}
