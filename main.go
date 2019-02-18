package main

import (
    "flag"
    "fmt"
    "github.com/gorilla/websocket"
    "github.com/mitsutoshi/bitflyergo"
    "github.com/mitsutoshi/bitflyerspider/helpers"
    "log"
    "os"
    "os/signal"
)

const (
    SymbolFXBTCJPY = "FX_BTC_JPY"
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
    go wsclient.Receive(brdSnpCh, brdCh, exeCh)

    mode := "csv"
    var executions []bitflyergo.Execution
    if *executionOpt {

        // websocketの約定履歴チャンネルの購読を開始する
        wsclient.SubscribeExecutions()

        // 指定された出力先への出力を開始する
        if mode == "stdout" {
            go helpers.WriteExecutionsToStdout(&executions)
        } else if mode == "stderr" {
            go helpers.WriteExecutionsToStderr(&executions)
        } else {
            go helpers.WriteExecutionsToFile(&executions, "csv", true, bufferSize)
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
