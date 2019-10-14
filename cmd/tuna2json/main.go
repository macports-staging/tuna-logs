package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TunaLog struct {
	Time       string `json:"time"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Code       string `json:"code"`
	Size       int    `json:"size"`
	Agent      string `json:"agent"`
	Scheme     string `json:"scheme"`
	RemoteAddr string `json:"remote"`
}

const tunaLogRegex = `^(?P<remote>[^ ]*) - (?P<user>[^ ]*) \[(?P<time>[^\]]*)\] "(?P<method>\S+)(?: +(?P<path>[^\"]*?)(?: +\S*)?)?" (?P<code>[^ ]*) (?P<size>[^ ]*) "(?P<type>[^\"]*)" "(?P<referer>[^\"]*)" "(?P<agent>[^\"]*)" - (?P<scheme>[^ ]*)$`

var tunaLogSpec = regexp.MustCompile(tunaLogRegex)

func main() {
	var writerWaitGroup sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	parseChan := make(chan string, 100)
	writeChan := make(chan []byte, 100)

	writerWaitGroup.Add(1)
	go writer(ctx, cancel, &writerWaitGroup, writeChan)

	const workerNum = 16
	var parserWaitGroup sync.WaitGroup
	parserWaitGroup.Add(workerNum)
	for i := 0; i < workerNum; i++ {
		go parseLines(ctx, &parserWaitGroup, parseChan, writeChan)
	}

	in := bufio.NewScanner(os.Stdin)
ScanLoop:
	for in.Scan() {
		select {
		case <-ctx.Done():
			break ScanLoop
		default:
			line := in.Text()
			parseChan <- line
		}
	}

	close(parseChan)
	parserWaitGroup.Wait()
	close(writeChan)
	writerWaitGroup.Wait()
}

func writer(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, logEntries <-chan []byte) {
WriteLoop:
	for {
		select {
		case entry := <-logEntries:
			if entry == nil {
				break WriteLoop
			}
			_, err := os.Stdout.Write(entry)
			if err != nil {
				log.Fatal(err)
				cancel()
				break WriteLoop
			}
		case <-ctx.Done():
			break WriteLoop
		}
	}

	wg.Done()
}

func parseLines(ctx context.Context, wg *sync.WaitGroup, lines <-chan string, output chan<- []byte) {
ParseLoop:
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				break ParseLoop
			}
			fields := tunaLogSpec.FindStringSubmatch(line)
			if len(fields) == 0 {
				continue
			}

			if !strings.HasPrefix(fields[5], "/macports/") {
				continue
			}
			method := fields[4]
			if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
				continue
			}

			timestamp, err := time.Parse("02/Jan/2006:15:04:05 -0700", fields[3])
			if err != nil {
				log.Printf("parse time failed: %s", line)
				continue
			}

			size, err := strconv.Atoi(fields[7])
			if err != nil {
				log.Printf("parse size failed: %s", line)
				continue
			}

			encoded, err := json.Marshal(&TunaLog{
				Time:       timestamp.UTC().Format(time.RFC3339),
				Method:     method,
				Path:       fields[5],
				Code:       fields[6],
				Size:       size,
				Agent:      fields[10],
				Scheme:     fields[11],
				RemoteAddr: fields[1],
			})
			if err != nil {
				log.Printf("json marshal failed: %s", err.Error())
				continue
			}
			encoded = append(encoded, '\n')
			output <- encoded
		case <-ctx.Done():
			break ParseLoop
		}
	}

	wg.Done()
}
