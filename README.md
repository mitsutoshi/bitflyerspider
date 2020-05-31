# bitflyerspider

bitflyerspider is tool collecting delivered data from bitflyer websoket. received data can be stored to the BigQuery or file.

websocket channel is coresponded only `lightning_FX_BTC_JPY`.

## How to use

1. Write config file (see config.toml).

2. Execute bitflyerspider

```
./bitflyerspider -c config.toml
```

## BigQuery fields structure

##### executions

|Name|Value|
|---|---|
|id|execution id|
|exec_date|executed date and time (UTC)|
|price|executed price|
|size|executed size (BTC)|
|side|taker side (BUY/SELL)|
|buy_child_order_acceptance_id|buy order id|
|sell_child_order_acceptance_id|sell order id|
|delay|time from execution to receipt|
|received_time|received time|

##### boards

|name|type|nullable|
|---|---|---|
|time|TIMESTAMP|NULLABLE|
|best_ask_price|INTEGER|NULLABLE|
|best_ask_size|FLOAT|NULLABLE|
|best_bid_price|INTEGER|NULLABLE|
|best_bid_size|FLOAT|NULLABLE|
|mid_price|INTEGER|NULLABLE|
|spread|INTEGER|NULLABLE|
