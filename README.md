# bitflyerspider

## Usage

Options

```
$ ./bin/bitflyerspider -h
Usage of ./bin/bitflyerspider:
  -b	Acquire board.
  -e	Acquire execution.
  -o string
    	File destination directory path. (default "./")
  -v	Show version info.
```

## 出力フォーマット

### 約定履歴

|項目名|出力内容|
|---|---|
|id|約定ID|
|exec_date|約定日時（UTF）|
|price|約定価格|
|size|約定サイズ（BTC）|
|side|テイク方向（BUY/SELL）|
|buy_child_order_acceptance_id|買い注文ID|
|sell_child_order_acceptance_id|売り注文ID|
|delay|受信遅延時間（秒）|

### 板
