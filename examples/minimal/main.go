package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/metrics"
	"time"

	"github.com/anderspitman/lightweaver-go"
)

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		lightweaver.Emit("chan", 1)
	})

	go func() {

		samples := []metrics.Sample{
			metrics.Sample{
				Name: "/sched/goroutines:goroutines",
			},
			metrics.Sample{
				Name: "/gc/heap/allocs:bytes",
			},
		}

		for {
			metrics.Read(samples)
			lightweaver.SetGauge("/num_goroutines", float64(samples[0].Value.Uint64()))
			mem := float64(samples[1].Value.Uint64()) / 1024 / 1024
			lightweaver.SetGauge("/mem_usage_mib", mem)
			time.Sleep(100 * time.Millisecond)
		}

	}()

	fmt.Println("Running")
	http.ListenAndServe(":3000", mux)
}

func printJson(data interface{}) {
	d, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(d))
}
