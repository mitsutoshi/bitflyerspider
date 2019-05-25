package bitflyerspider

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

const filePermission os.FileMode = 0644

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
    file, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, filePermission)
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

            if time.Now().Truncate(time.Hour * 12).After(today.Truncate(time.Hour * 12)) {

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

                file, err = os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, filePermission)
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
    fId, _ := t.FieldByName("Id")
    fExecDate, _ := t.FieldByName("ExecDate")
    fSide, _ := t.FieldByName("Side")
    fPrice, _ := t.FieldByName("Price")
    fSize, _ := t.FieldByName("Size")
    fBuyChildOrderAcceptanceId, _ := t.FieldByName("BuyChildOrderAcceptanceId")
    fSellChildOrderAcceptanceId, _ := t.FieldByName("SellChildOrderAcceptanceId")
    fDelay, _ := t.FieldByName("Delay")
    return fmt.Sprintf("%s%c%s%c%s%c%s%c%s%c%s%c%s%c%v",
        fId.Tag.Get("json"), delimiter,
        fExecDate.Tag.Get("json"), delimiter,
        fSide.Tag.Get("json"), delimiter,
        fPrice.Tag.Get("json"), delimiter,
        fSize.Tag.Get("json"), delimiter,
        fBuyChildOrderAcceptanceId.Tag.Get("json"), delimiter,
        fSellChildOrderAcceptanceId.Tag.Get("json"), delimiter,
        fDelay.Tag.Get("json"))
}

// 約定履歴のCSV出力用文字列を返します
func ExecutionToCsv(e *bitflyergo.Execution) string {
    return fmt.Sprintf("%d,%s,%s,%d,%.8f,%s,%s,%v",
        e.Id, e.ExecDate, e.Side, int(e.Price), e.Size, e.BuyChildOrderAcceptanceId, e.SellChildOrderAcceptanceId, e.Delay.Seconds())
}

// 約定履歴のTSV出力用文字列を返します
func ExecutionToTsv(e *bitflyergo.Execution) string {
    return fmt.Sprintf("%d\t%s\t%s\t%d\t%.8f\t%s\t%s\t%v",
        e.Id, e.ExecDate, e.Side, int(e.Price), e.Size, e.BuyChildOrderAcceptanceId, e.SellChildOrderAcceptanceId, e.Delay.Seconds())
}

// 約定履歴のJSON出力用文字列を返します
func ExecutionToJson(e *bitflyergo.Execution) string {
    b, err := json.Marshal(e)
    if err != nil {
        log.Fatal(err)
    }
    return string(b) + "\n"
}
