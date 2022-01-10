package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

// Setup
var (
	fileTime        int
	chunksPerSecond int
	fileSize        int
)

func main() {
	flag.IntVar(&fileTime, "time", 3, "How many seconds should file be downloaded")
	flag.IntVar(&chunksPerSecond, "chunks", 10, "How many chunks per seconds to send")
	flag.IntVar(&fileSize, "size", 3*10*1024*1024, "Payload size of the each file") // 30MiB by default
	flag.Parse()

	// Prepare random (not good compressable) payload
	rand.Seed(0xdeadbeef)
	chunkSize := fileSize
	if fileTime != 0 {
		chunkSize = fileSize / fileTime / chunksPerSecond
		fileSize = chunkSize * fileTime * chunksPerSecond
	}
	payload := make([]byte, fileSize)
	rand.Read(payload)

	m := sync.Mutex{}
	cFiles := 0
	cBytes := 0

	router := http.NewServeMux()
	server := &http.Server{Addr: ":12345", Handler: router}

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			panic("expected http.ResponseWriter to be an http.Flusher")
		}

		if fileTime == 0 {
			w.Write(payload)
		} else {
			for i := 0; i < fileTime*chunksPerSecond; i++ {
				w.Write(payload[i*chunkSize : (i+1)*chunkSize])
				flusher.Flush()
				time.Sleep(time.Second / time.Duration(chunksPerSecond))
			}
		}

		m.Lock()
		cFiles++
		cBytes += fileSize
		m.Unlock()
	})

	// Test control (start/stop)
	var start time.Time
	router.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[ORIGIN] Starting\n")
		start = time.Now()
	})
	router.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[ORIGIN] Stopping\n")
		server.Shutdown(context.Background())
	})

	// Run server
	fmt.Printf(
		"[ORIGIN] Ready on localhost:12345 to serve files (file size: %d, file_time: %d, one chunk size: %d)\n",
		fileSize, fileTime, chunkSize,
	)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("Server error: %v", err))
	}

	// After server.Shutdown...
	totalTime := time.Now().Sub(start)
	avgMbps := float64(cBytes) * 8 / totalTime.Seconds() / 1_000_000
	fmt.Printf(
		"[ORIGIN] requests: %d, bytes: %d, totalTime: %v, avgMbps: %f\n",
		cFiles, cBytes, totalTime, avgMbps,
	)

	f, err := os.OpenFile("origin_stats.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	fmt.Fprintf(f, "%d;%d;%f\n", cFiles, cBytes, avgMbps)
}
