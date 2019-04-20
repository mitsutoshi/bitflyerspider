package main

import (
    "cloud.google.com/go/bigquery"
    //"cloud.google.com/go/civil"
    "context"
    "flag"
    "fmt"
    "github.com/gorilla/websocket"
    "github.com/mitsutoshi/bitflyergo"
    "github.com/mitsutoshi/bitflyerspider/helpers"
    "log"
    "os"
    "os/signal"
    "time"
)

const (
    SymbolFXBTCJPY = "FX_BTC_JPY"
)

// Write mode
const (
    modeStdout   = "stdout"
    modeStderr   = "stderr"
    modeCsv      = "csv"
    modeBigQuery = "bigquery"
)

var (
    version      string
    revision     string
    outOpt       = flag.String("o", "./", "File destination directory path.")
    executionOpt = flag.Bool("e", false, "Acquire execution.")
    boardOpt     = flag.Bool("b", false, "Acquire board.")
    verOpt       = flag.Bool("v", false, "Show version info.")
    bufferSize   = 1000
)

func main() {

    // オプションを解析
    flag.Parse()

    // バージョン情報を出力
    if *verOpt {
        fmt.Printf("%s (rev %s)\n", version, revision)
        os.Exit(0)
    } else if !*executionOpt && !*boardOpt {
        fmt.Println("You specified illegal option. Use -h option.")
        os.Exit(1)
    }

    log.Println("File destination directory:", *executionOpt)
    log.Println("Acquire execution:", *executionOpt)
    log.Println("Acquire board:", *executionOpt)

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

    mode := modeBigQuery
    var executions []bitflyergo.Execution
    if *executionOpt {

        // websocketの約定履歴チャンネルの購読を開始する
        wsclient.SubscribeExecutions()

        // 指定された出力先への出力を開始する
        if mode == modeStdout {
            go helpers.WriteExecutionsToStdout(&executions)
        } else if mode == modeStderr {
            go helpers.WriteExecutionsToStderr(&executions)
        } else if mode == modeCsv {
            go helpers.WriteExecutionsToFile(&executions, "csv", true, bufferSize)
        } else if mode == modeBigQuery {

            go func() {

                interval := 15 * time.Second
                ctx := context.Background()
                bqClient, err := bigquery.NewClient(ctx, "catbot")
                if err != nil {
                    log.Fatal(err)
                }
                inserter := bqClient.Dataset("bitflyer").Table("executions_all").Inserter()

                for {
                    to := len(executions)

                    if to > 0 {

                        // BigQueryへ登録するための型へ変換する
                        var items []*helpers.Execution
                        for i := 0; i < to; i++ {
                            items = append(items, &helpers.Execution{
                                Id:                         executions[i].Id,
                                ExecDate:                   executions[i].ExecDate,
                                Price:                      executions[i].Price,
                                Size:                       executions[i].Size,
                                Side:                       executions[i].Side,
                                BuyChildOrderAcceptanceId:  executions[i].BuyChildOrderAcceptanceId,
                                SellChildOrderAcceptanceId: executions[i].SellChildOrderAcceptanceId,
                                Delay:                      executions[i].Delay.Seconds(),
                            })
                        }

                        // Insert
                        if err := inserter.Put(ctx, items); err != nil {
                            log.Println(err, "data ->", items)
                            continue
                        }

                        // Insertした要素を削除
                        executions = executions[to:]
                        log.Printf(" Finished write %v executions to BigQuery.\n", len(executions))
                    }
                    time.Sleep(interval)
                }
            }()

        } else {
            panic(fmt.Sprintf("Unkown mode '%v'", mode))
        }

    }

    var boards []bitflyergo.Board
    if *boardOpt {

        // websocketの板情報チャンネルの購読を開始する
        wsclient.SubscribeBoardSnapshot()
        wsclient.SubscribeBoard()

        // 指定された出力先への出力を開始する
        if mode == "stdout" {
            go helpers.WriteBoardToStdout(&boards)
        } else if mode == "stderr" {
            go helpers.WriteBoardToStderr(&boards)
        } else {
            go helpers.WriteBoardsFile(&boards, bufferSize)
        }
    }

    for {

        select {

        case board := <-brdSnpCh: // 板情報スナップショット受信
            boards = append(boards, board)

            // 一度目を受信した後は差分情報のみで板を更新するため、snapshotはUnsubscribeする
            wsclient.UnsubscribeBoardSnapshot()
            break

        case board := <-brdCh: // 板情報受信
            boards = append(boards, board)
            break

        case execution := <-exeCh: // 約定履歴受信
            executions = append(executions, execution...)
            break

        case s := <-interrupt: // シグナル受信
            fmt.Println("Interrupt ", s)
            err := wsclient.Con.WriteMessage(
                websocket.CloseMessage,
                websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
            if err != nil {
                fmt.Errorf("Close Error!", err)
                return
            }
            return
        }
    }
}
