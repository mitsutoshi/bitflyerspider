package helpers

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "reflect"
    "sync"
    "time"

    "github.com/mitsutoshi/bitflyergo"
)

func WriteExecutionsToStdout(executions *[]bitflyergo.Execution) {
    WriteExecutionsTo(executions, os.Stdout)
}

func WriteExecutionsToStderr(executions *[]bitflyergo.Execution) {
    WriteExecutionsTo(executions, os.Stderr)
}

func WriteExecutionsTo(executions *[]bitflyergo.Execution, writer *os.File) {
    var mu sync.Mutex
    for {
        if len(*executions) >= 1 {
            mu.Lock()
            for _, e := range *executions {
                fmt.Fprintln(writer, ExecutionToTsv(&e))
            }
            *executions = nil
            mu.Unlock()
        }
        time.Sleep(100 * time.Millisecond)
    }
}

func WriteBoardToStdout(boards *[]bitflyergo.Board) {
    WriteBoardTo(boards, os.Stdout)
}

func WriteBoardToStderr(boards *[]bitflyergo.Board) {
    WriteBoardTo(boards, os.Stderr)
}

func WriteBoardTo(boards *[]bitflyergo.Board, writer *os.File) {
    var mu sync.Mutex
    for {
        if len(*boards) >= 1 {
            mu.Lock()
            for _, b := range *boards {
                fmt.Fprintln(writer, BoardToJson(&b))
            }
            *boards = nil
            mu.Unlock()
        }
        time.Sleep(100 * time.Millisecond)
    }
}

func WriteExecutionsToFile(executions *[]bitflyergo.Execution, fileType string, headerOn bool, bufferSize int) {

    prefix := "executions"

    // 保存先とするファイルの名前を取得
    today := time.Now()
    name := createFileName(prefix, today, fileType)

    // ファイルの有無をチェック
    exist := false
    if _, err := os.Stat(name); err == nil {
        exist = true
    }

    // 約定履歴を書き込むためのファイルを開く
    file, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
    log.Println("File open.", name)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    // ヘッダ書き込み
    if !exist && headerOn {
        writeHeaders(file, fileType)
    }

    var mu sync.Mutex
    for {
        if len(*executions) > bufferSize {

            if time.Now().Truncate(time.Minute * 60).After(today.Truncate(time.Minute * 60)) {

                // ファイルを切り替えるためオープン中のファイルはクローズする
                log.Println("File close.", name)
                file.Close()
                file.Sync()

                // 新しいファイルをオープン
                today = time.Now()
                name = createFileName(prefix, today, fileType)

                // ファイルの有無をチェックして同名ファイルが存在する場合は削除
                if _, err := os.Stat(name); err == nil {
                    if err := os.Remove(name); err != nil {
                        log.Fatal(err)
                    }
                }

                file, err = os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
                log.Println("File open.", name)
                if err != nil {
                    log.Fatal(err)
                }
                defer file.Close()

                // ヘッダ書き込み
                writeHeaders(file, fileType)
            }

            mu.Lock()
            for _, e := range *executions {
                switch fileType {
                case "csv":
                    fmt.Fprintln(file, ExecutionToCsv(&e))
                case "tsv":
                    fmt.Fprintln(file, ExecutionToTsv(&e))
                case "json":
                    fmt.Fprintln(file, ExecutionToJson(&e))
                }
            }
            *executions = nil
            mu.Unlock()
            file.Sync()
        }
        time.Sleep(1 * time.Second)
    }
}

// 約定履歴を出力するファイル名を返します。
func createFileName(prefix string, day time.Time, fileType string) string {
    var ext string
    if fileType == "tsv" {
        ext = ".tsv"
    } else if fileType == "csv" {
        ext = ".csv"
    } else if fileType == "json" {
        ext = ".json"
    } else {
        ext = ".txt"
    }
    date := day.Format("20060102030405")
    return prefix + "_" + date + ext
}

func writeHeaders(file *os.File, fileType string) {
    if fileType == "csv" {
        fmt.Fprintln(file, getExecutionHeaders(','))
    } else if fileType == "tsv" {
        fmt.Fprintln(file, getExecutionHeaders('\t'))
    }
}

func getExecutionHeaders(delimiter rune) string {
    t := reflect.TypeOf(bitflyergo.Execution{Price: 1})
    return fmt.Sprintf("%s%c%s%c%s%c%s%c%s%c%s%c%s%c%v",
        t.Field(0).Tag.Get("json"), delimiter,
        t.Field(1).Tag.Get("json"), delimiter,
        t.Field(2).Tag.Get("json"), delimiter,
        t.Field(3).Tag.Get("json"), delimiter,
        t.Field(4).Tag.Get("json"), delimiter,
        t.Field(5).Tag.Get("json"), delimiter,
        t.Field(6).Tag.Get("json"), delimiter,
        t.Field(7).Tag.Get("json"))
}

//type JsonFile struct {
//    Data       *[]string
//    BufferSize int
//}

//
//func (pipe BoardStdOutPipe) Output() {
//   var mu sync.Mutex
//   for {
//       if len(pipe.Rows) >= 1 {
//           mu.Lock()
//           for _, row := range pipe.Rows {
//               fmt.Print(row.(Row).TsvRow())
//           }
//           pipe.Rows = nil
//           mu.Unlock()
//       }
//       time.Sleep(100 * time.Millisecond)
//   }
//}
//
//type BoardFilePipe struct {
//    Boards Board
//}

// 約定履歴のCSV出力用文字列を返します
func ExecutionToCsv(e *bitflyergo.Execution) string {
    return fmt.Sprintf("%d,%s,%s,%d,%f,%s,%s,%v",
        e.Id, e.ExecDate, e.Side, int(e.Price), e.Size, e.BuyChildOrderAcceptanceId, e.SellChildOrderAcceptanceId, e.Delay)
}

// 約定履歴のTSV出力用文字列を返します
func ExecutionToTsv(e *bitflyergo.Execution) string {
    return fmt.Sprintf("%d\t%s\t%s\t%d\t%f\t%s\t%s\t%v",
        e.Id, e.ExecDate, e.Side, int(e.Price), e.Size, e.BuyChildOrderAcceptanceId, e.SellChildOrderAcceptanceId, e.Delay)
}

// 約定履歴のJSON出力用文字列を返します
func ExecutionToJson(e *bitflyergo.Execution) string {
    b, err := json.Marshal(e)
    if err != nil {
        log.Fatal(err)
    }
    return string(b) + "\n"
}

func WriteBoardsFile(boards *[]bitflyergo.Board, bufferSize int) {

    prefix := "boards"

    // 保存先とするファイルの名前を取得
    today := time.Now()
    name := createFileName(prefix, today, "json")

    file, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
    log.Println("File open.", name)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    var mu sync.Mutex
    for {
        if len(*boards) > bufferSize {

            if time.Now().Truncate(time.Minute * 60).After(today.Truncate(time.Minute * 60)) {

                // ファイルを切り替えるためオープン中のファイルはクローズする
                log.Println("File close.", name)
                file.Close()
                file.Sync()

                // 新しいファイルをオープン
                today = time.Now()
                name = createFileName(prefix, today, "json")

                // ファイルの有無をチェックして同名ファイルが存在する場合は削除
                if _, err := os.Stat(name); err == nil {
                    if err := os.Remove(name); err != nil {
                        log.Fatal(err)
                    }
                }

                file, err = os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
                log.Println("File open.", name)
                if err != nil {
                    log.Fatal(err)
                }
                defer file.Close()
            }

            mu.Lock()
            for _, b := range *boards {
				fmt.Fprintln(file, BoardToJson(&b))
            }
            *boards = nil
            mu.Unlock()
            file.Sync()
        }
        time.Sleep(1 * time.Second)
    }
}

// 板のJSON出力用文字列を返します
func BoardToJson(b *bitflyergo.Board) string {
    data, err := json.Marshal(b)
    if err != nil {
        log.Fatal(err)
    }
    return string(data)
}
