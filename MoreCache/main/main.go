package main

import (
	"MoreCache"
	. "MoreCache"
	"flag"
	"fmt"
	"log"
	"net/http"
)


var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func test_server() {
	NewGroup("scores", 2 << 10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		},
	))
	addr := "localhost:9999"
	peers := NewHTTPPool(addr)
	log.Println("morecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
func createGroup() *MoreCache.Group {
	return MoreCache.NewGroup("sores", 2 << 10, MoreCache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		},
	))
}
func startCacheServer(addr string, addrs []string, m *MoreCache.Group) {
	peers := MoreCache.NewHTTPPool(addr)
	peers.Set(addrs...)
	m.RegisterPeers(peers)
	log.Println("MoreCache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
	// h t t p : / / = 7
}
func startAPIServer(apiAddr string, m *MoreCache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := m.Get(key)
			if err != nil {
				http.Error(w, err.Error(),http.StatusInternalServerError)
				return 
			}	
			w.Header().Set("Content-Type", "application/octet-stram")
			w.Write(view.ByteSlice())
		},
	))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil ))
}
func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "MoreCache server port")
	flag.BoolVar(&api, "api", false, "Start api server?")
	flag.Parse()
	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}
	m := createGroup()
	if api {
		go startAPIServer(apiAddr, m)
	}
	startCacheServer(addrMap[port], []string(addrs), m)
}


