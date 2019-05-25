package main

import (
    "cloud.google.com/go/bigquery"
    "context"
    "flag"
    "fmt"
    "github.com/BurntSushi/toml"
    "github.com/gorilla/websocket"
    "github.com/mitsutoshi/bitflyergo"
    bitflyerspider "github.com/mitsutoshi/bitflyerspider/lib"
    "github.com/mitsutoshi/bitflyerspider/lib/bq"
    "io"
    "log"
    "os"
    "os/signal"
    "time"
)

type Config struct {
    DryRun    bool   `toml:"dryrun"`
    Dest      string `toml:"dest"`
    Execution bool   `toml:"execution"`
    Board     bool   `toml:"board"`
    BigQuery  BigQueryConfig
}

type BigQueryConfig struct {
    Project             string `toml:"project"`
    Dataset             string `toml:"dataset"`
    ExecutionsTable     string `toml:"executionsTable"`
    BoardsTable         string `toml:"boardsTable"`
    CredentialsFilePath string `toml:credentialsFilePath`
}

const (
    SymbolFXBTCJPY         = "FX_BTC_JPY"
    modeCsv                = "csv"
    modeBigQuery           = "bigquery"
    logFileName            = "application.log"
    gAppCredentialsEnvName = "GOOGLE_APPLICATION_CREDENTIALS"
)

// 起動オプション
var (
    verOpt  = flag.Bool("v", false, "Show version info.")
    confOpt = flag.String("c", "./config.toml", "Config file path.")
)

var (
    version    string
    revision   string
    bufferSize = 1000
    config     Config
    collector  = bitflyerspider.NewCollector()
    bqClient   *bigquery.Client
    ctx        = context.Background()
    running    = true
)

func main() {

    // オプションを解析
    flag.Parse()

    // バージョン情報を出力
    if *verOpt {
        fmt.Printf("%s (rev %s)\n", version, revision)
        os.Exit(0)
    }

    // ログファイルのセットアップ
    logfile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
    if err != nil {
        log.Panicf("Cannot open %v: %v\n", logfile, err.Error())
    }
    defer logfile.Close()
    log.SetOutput(io.MultiWriter(logfile, os.Stdout)) // ログをファイルと標準出力の両方へ出力するように指定
    log.SetFlags(log.Ldate | log.Ltime)

    // 設定ファイルのロード
    _, err = toml.DecodeFile(*confOpt, &config)
    if err != nil {
        panic(err)
    }
    log.Printf("Load config file: %s\n", *confOpt)
    log.Printf("Config -> dryrun=%v, dest=%v, execution=%v, board=%v\n",
        config.DryRun, config.Dest, config.Execution, config.Board)

    // 約定履歴の取得と板の取得の両方が無効化されている場合は終了
    if !config.Execution && !config.Board {
        log.Printf("Target is not specified.")
        os.Exit(1)
    }

    // GCPへ接続するための認証情報をセットアップ
    log.Printf("Destination: %s\n", config.Dest)
    if config.Dest == modeBigQuery {
        if os.Getenv(gAppCredentialsEnvName) == "" {
            os.Setenv(gAppCredentialsEnvName, config.BigQuery.CredentialsFilePath)
        }
        log.Printf("Destination of executions: %s.%s.%s\n",
            config.BigQuery.Project, config.BigQuery.Dataset, config.BigQuery.ExecutionsTable)
        log.Printf("Destination of boards: %s.%s.%s\n",
            config.BigQuery.Project, config.BigQuery.Dataset, config.BigQuery.BoardsTable)
    }

    // BigQueryクライアントを作成
    bqClient, err = bigquery.NewClient(ctx, config.BigQuery.Project)
    if err != nil {
        log.Fatal(err)
    }

    // シグナル受信の対応
    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt)

    // websocket受信処理を開始
    wsclient := bitflyergo.WebSocketClient{Symbol: SymbolFXBTCJPY}
    wsclient.Connect()
    defer wsclient.Con.Close()
    brdSnpCh := make(chan bitflyergo.Board)
    brdCh := make(chan bitflyergo.Board)
    exeCh := make(chan []bitflyergo.Execution)
    errCh := make(chan error)
    go wsclient.Receive(brdSnpCh, brdCh, exeCh, errCh)

    var executions []bitflyergo.Execution
    if config.Execution {

        // websocketの約定履歴チャンネルの購読を開始する
        wsclient.SubscribeExecutions()

        // 指定された出力先への出力を開始する
        if config.Dest == modeCsv {
            go bitflyerspider.WriteExecutionsToFile(&executions, "csv", true, bufferSize)
        } else if config.Dest == modeBigQuery {

            log.Println("Start to writing executions to BigQuery.")
            go func() {
                interval := 60 * time.Second
                inserter := bqClient.Dataset(config.BigQuery.Dataset).Table(config.BigQuery.ExecutionsTable).Inserter()

                var item *bq.BqExecution

                for {
                    to := len(executions)
                    if to > 0 {

                        // BigQueryへ登録するための型へ変換する
                        var items []*bq.BqExecution
                        for i := 0; i < to; i++ {

                            // 新しい約定履歴が、itemとマージ可能かチェック
                            if item != nil && executions[i].Side == item.Side && executions[i].Price == item.Price &&
                                ((item.Side == "BUY" && executions[i].BuyChildOrderAcceptanceId == item.BuyChildOrderAcceptanceId) ||
                                    (item.Side == "SELL" && executions[i].SellChildOrderAcceptanceId == item.SellChildOrderAcceptanceId)) {

                                // 可能ならサイズをマージする。
                                // 約定日時、ID、買い注文ID、売り注文ID、遅延秒数は１件目のデータのものを採用する、時間などはほぼずれないので問題ない
                                item.Size += executions[i].Size
                                //fmt.Printf("Mrg item %v, side=%v, size=%v\n", item.Id, item.Side, item.Size)

                            } else {

                                // マージできないため、itemを区切る
                                if item != nil {
                                    items = append(items, item)
                                    //fmt.Printf("Apd item %v, side=%v, size=%v\n", item.Id, item.Side, item.Size)
                                }

                                // 新たな約定履歴のitemを作成
                                item = &bq.BqExecution{
                                    Id:                         executions[i].Id,
                                    ExecDate:                   executions[i].ExecDate,
                                    Price:                      executions[i].Price,
                                    Size:                       executions[i].Size,
                                    Side:                       executions[i].Side,
                                    BuyChildOrderAcceptanceId:  executions[i].BuyChildOrderAcceptanceId,
                                    SellChildOrderAcceptanceId: executions[i].SellChildOrderAcceptanceId,
                                    Delay:                      executions[i].Delay.Seconds(),
                                }
                                //fmt.Printf("New item %v, side=%v, size=%v\n", item.Id, item.Side, item.Size)
                            }
                        }

                        // Insert
                        if !config.DryRun {
                            if err := inserter.Put(ctx, items); err != nil {
                                log.Println(err, "data ->", items)
                                continue
                            }
                        }

                        // Insertした要素を削除
                        executions = executions[to:]
                        log.Printf(" Finished write %v executions to BigQuery.\n", len(executions))
                    }
                    time.Sleep(interval)
                }
            }()

        } else {
            panic(fmt.Sprintf("Unkown mode '%v'", config.Dest))
        }
    }

    if config.Board {

        // 購読開始
        wsclient.SubscribeBoardSnapshot()
        wsclient.SubscribeBoard()

        // 集計開始
        go collector.Agg()

        // BigQueryへBoardsを登録
        log.Println("Start to writing boards to BigQuery.")
        go writeBoardBigQuery()
    }

    for {
        select {
        case boardSnap := <-brdSnpCh: // 板情報スナップショット受信
            collector.UpdateBoard(&boardSnap, true)

            // 一度目を受信した後は差分情報のみで板を更新するため、snapshotはUnsubscribeする
            wsclient.UnsubscribeBoardSnapshot()
            break

        case board := <-brdCh: // 板情報受信
            collector.UpdateBoard(&board, false)
            break

        case execution := <-exeCh: // 約定履歴受信
            executions = append(executions, execution...)
            break

        case s := <-interrupt: // シグナル受信
            fmt.Println("Interrupt ", s)
            running = false
            wsclient.Con.WriteMessage(
                websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
            return
        }
    }
}

func writeBoardBigQuery() {

    const interval = 5 * time.Second // Boardの単位時間ごとのサマリの保持件数をチェックする間隔
    const threhold = 60 * 5          // BigQueryへの登録処理を実行するサマリの件数（900秒分 = 15分ごと）

    // Boardテーブルのinserter
    inserter := bqClient.Dataset(config.BigQuery.Dataset).Table(config.BigQuery.BoardsTable).Inserter()

    for running {

        // 板情報が一定件数が溜まるごとにBigQueryへ登録する
        if len(collector.SummaryPerSec) >= threhold {
            i := len(collector.SummaryPerSec)
            items := collector.SummaryPerSec[:i]

            if !config.DryRun {
                err := inserter.Put(ctx, items)
                if err != nil {
                    log.Println(err, "data ->", items)
                } else {

                    // 保存が完了したBoardはitemsから削除
                    log.Printf(" Finished write %v boards to BigQuery.\n", len(items))
                    collector.SummaryPerSec = collector.SummaryPerSec[i:]
                }
            }
        }
        time.Sleep(interval)
    }
}
