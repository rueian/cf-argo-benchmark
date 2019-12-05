package main

import (
	"context"
	"contrib.go.opencensus.io/exporter/prometheus"
	"crypto/tls"
	"encoding/json"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var (
	Latency     = stats.Float64("latency", "http latency", "ms")
	KeyPath, _  = tag.NewKey("path")
	KeyHost, _  = tag.NewKey("host")
	KeyCode, _  = tag.NewKey("code")
	LatencyView = &view.View{
		Name:        "latency",
		Measure:     Latency,
		Description: "The distribution of the latencies",
		Aggregation: view.Distribution(0, 25, 50, 75, 100, 200, 400, 600, 800, 1000, 2000, 4000, 6000),
		TagKeys:     []tag.Key{KeyPath, KeyHost, KeyCode},
	}
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type Config struct {
	Endpoints map[string]int
	endpoints []string
	Hosts     []string
	UserAgent string
	Sleep     int
}

var mu sync.RWMutex
var config Config

func setup() {
	bs, err := ioutil.ReadFile(env("CONFIG", "/client/config.json"))
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(bs, &config); err != nil {
		panic(err)
	}

	config.endpoints = []string{}
	for k, v := range config.Endpoints {
		for i := 0; i < v; i++ {
			config.endpoints = append(config.endpoints, k)
		}
	}
}

func main() {

	rand.Seed(time.Now().UnixNano())

	pe, _ := prometheus.NewExporter(prometheus.Options{Namespace: "client"})

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		if err := http.ListenAndServe(":8888", mux); err != nil {
			panic(err)
		}
	}()

	view.RegisterExporter(pe)

	if err := view.Register(LatencyView); err != nil {
		panic(err)
	}

	setup()

	go func() {
		for {
			mu.Lock()
			setup()
			mu.Unlock()
			time.Sleep(5 * time.Second)
		}
	}()

	workers, err := strconv.Atoi(env("WORKERS", "1"))
	if err != nil {
		panic(err)
	}

	for i := 0; i < workers; i++ {
		go func() {
			var (
				cc     Config
				client http.Client
				tick   <-chan time.Time
			)

			transport := *http.DefaultTransport.(*http.Transport)
			transport.ForceAttemptHTTP2 = true
			transport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}

			init := func() {
				mu.RLock()
				cc = config
				mu.RUnlock()

				client = http.Client{
					Transport: &transport,
				}

				tick = time.Tick(10 * time.Second)
			}

			init()

			for {
				select {
				case <-tick:
					init()
				default:
				}

				endpoint := cc.endpoints[rand.Intn(len(cc.endpoints))]
				host := cc.Hosts[rand.Intn(len(cc.Hosts))]

				req, err := http.NewRequest("GET", host+endpoint, nil)
				if err != nil {
					panic(err)
				}
				if cc.UserAgent != "" {
					req.Header.Set("User-Agent", cc.UserAgent)
				}

				var ts = time.Now()
				var code string

				if resp, err := client.Do(req); err != nil {
					log.Println(err)
					code = "5xx"
				} else {
					ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					code = strconv.Itoa(resp.StatusCode)
				}

				ctx, _ := tag.New(context.Background(), tag.Insert(KeyPath, endpoint), tag.Insert(KeyHost, host), tag.Insert(KeyCode, code))
				stats.Record(ctx, Latency.M(float64(time.Since(ts).Nanoseconds())/1e6))

				if cc.Sleep > 0 {
					time.Sleep(time.Duration(cc.Sleep) * time.Millisecond)
				}
			}
		}()
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
