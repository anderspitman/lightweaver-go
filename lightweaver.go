package lightweaver

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
)

type Server struct {
	mut     *sync.Mutex
	clients map[string]map[string]*Client
	gauges  map[string]float64
}

type Client struct {
	ch chan any
}

func NewServer() *Server {

	s := &Server{
		mut:     new(sync.Mutex),
		clients: make(map[string]map[string]*Client),
		gauges:  make(map[string]float64),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		channel := r.URL.Path

		id, err := genRandomKey()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		ch := make(chan any)

		s.mut.Lock()
		_, exists := s.clients[channel]
		if !exists {
			s.clients[channel] = make(map[string]*Client)
		}

		s.clients[channel][id] = &Client{
			ch: ch,
		}

		// Always send current value upon connection
		if val, exists := s.gauges[channel]; exists {
			go func() {
				ch <- val
			}()
		}
		s.mut.Unlock()

		printJson(s.clients)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/octet-stream")

	LOOP:
		for {
			select {
			case msg := <-ch:
				bytes, err := json.Marshal(msg)
				if err != nil {
					continue
				}

				body := append(bytes, '\n')

				w.Write(body)

				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			case <-r.Context().Done():
				break LOOP
			}
		}

		s.mut.Lock()
		delete(s.clients[channel], id)
		s.mut.Unlock()

		printJson(s.clients)
	})

	go http.ListenAndServe(":9007", mux)

	return s
}

func (s *Server) SetGauge(channel string, val float64) {
	s.mut.Lock()
	defer s.mut.Unlock()

	prev, exists := s.gauges[channel]
	if exists && prev != val {
		for _, client := range s.clients[channel] {
			client.ch <- val
		}
	}

	s.gauges[channel] = val
}

func (s *Server) Emit(channel string, val any) (err error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	for _, client := range s.clients[channel] {
		client.ch <- val
	}

	return
}

var server *Server

func init() {
	server = NewServer()
}

func Emit(channel string, val any) {
	server.Emit(channel, val)
}

func SetGauge(channel string, val float64) {
	server.SetGauge(channel, val)
}

func genRandomKey() (string, error) {
	const chars string = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	id := ""
	for i := 0; i < 32; i++ {
		randIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		id += string(chars[randIndex.Int64()])
	}
	return id, nil
}

func printJson(data interface{}) {
	d, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(d))
}
