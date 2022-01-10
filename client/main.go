package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	baseURL   string
	originURL string
	paralel   int
	runTime   int
)

func main() {
	flag.StringVar(&baseURL, "url", "http://127.0.0.1:1080/file-", "Base URL, number will be appended")
	flag.StringVar(&originURL, "origin", "http://127.0.0.1:12345/", "URL used to communicate with the test origin")
	flag.IntVar(&paralel, "p", 100, "How many paralel request to fire")
	flag.IntVar(&runTime, "t", 30, "How many second to run the")
	flag.Parse()
	timeLimit := time.Now().Add(time.Second * time.Duration(runTime))

	fmt.Printf("[CLIENT] Starting test with %d paralel requests for %d seconds\n", paralel, runTime)

	wg := sync.WaitGroup{}

	// Global counters
	m := sync.Mutex{}
	gStart := time.Now()
	gcRequests := 0
	gcRequestMap := map[int]int{}
	gcBytes := 0
	gcByteSeconds := 0.0
	gcTimeToFirstByte := time.Duration(0)

	http.Post(originURL+"start", "", nil)
	for worker := 0; worker < paralel; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()

			// Per worker counters
			buffer := make([]byte, 1024)
			cRequests := 0
			cRequestsMap := map[int]int{}
			cBytes := 0
			cByteSeconds := 0.0
			cTimeToFirstByte := time.Duration(0)

			i := 0
			for time.Now().Before(timeLimit) {
				url := baseURL + strconv.Itoa(i)
				i++
				start := time.Now()
				resp, err := http.Get(url)
				if err != nil {
					fmt.Printf("[CLIENT] [%d-%d] ERROR: %v, resp: %v\n", worker, i, err, resp)
					continue
				}

				first := true
				for {
					n, err := resp.Body.Read(buffer)
					if n > 0 {
						if first {
							first = false
							cTimeToFirstByte += time.Now().Sub(start)
						}
						cBytes += n
						cByteSeconds += float64(n) * time.Now().Sub(start).Seconds()
					}
					if err == io.EOF {
						break
					} else if err != nil {
						fmt.Printf("[CLIENT] [%d-%d] READ ERROR: %v\n", worker, i, err)
						break
					}
				}
				resp.Body.Close()
				cRequests++
				c, _ := cRequestsMap[resp.StatusCode]
				cRequestsMap[resp.StatusCode] = c + 1
			}

			// Add to global counters
			m.Lock()
			gcRequests += cRequests
			gcBytes += cBytes
			gcByteSeconds += cByteSeconds
			gcTimeToFirstByte += cTimeToFirstByte
			for k, c := range cRequestsMap {
				cc, _ := gcRequestMap[k]
				gcRequestMap[k] = cc + c
			}
			m.Unlock()
		}(worker)
	}
	wg.Wait()
	http.Post(originURL+"stop", "", nil)

	totalTime := time.Now().Sub(gStart)
	avgMbps := float64(gcBytes) * 8 / totalTime.Seconds() / 1_000_000
	avgByteWaitTime := gcByteSeconds / float64(gcBytes)
	avgTimeToFirstByte := gcTimeToFirstByte.Seconds() / float64(gcRequests)
	fmt.Printf(
		"[CLIENT] requests: %d (status: %v), bytes: %d, totalTime: %v, avgMbps: %f, avgByteWaitTime: %f, avgTimeToFirstByte: %f\n",
		gcRequests, gcRequestMap, gcBytes, totalTime,
		avgMbps, avgByteWaitTime, avgTimeToFirstByte,
	)

	f, err := os.OpenFile("client_stats.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	fmt.Fprintf(f, "%d;%d;%f;%f;%f\n", gcRequests, gcBytes, avgMbps, avgByteWaitTime, avgTimeToFirstByte)
}
