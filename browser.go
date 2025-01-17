//go:build browser

package main

import (
	"bytes"
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/browser"
	"github.com/xmdhs/player-go/api"
	"github.com/xmdhs/player-go/cors"
	"go.etcd.io/bbolt"

	"github.com/mattn/go-ieproxy"
)

//go:embed frontend/dist
var assets embed.FS

func main() {
	cxt := context.Background()

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = ieproxy.GetProxyFunc()

	db, err := bbolt.Open("player.db", 0600, nil)
	if err != nil {
		panic(err)
	}

	cport := cors.Server(cxt, t)
	apiPort := api.Server(cxt, db, t)

	ieproxy.OverrideEnvWithStaticProxy()

	mux := http.NewServeMux()

	mux.HandleFunc("/", indexH(cport, apiPort))
	mux.Handle("/assets/", AddPrefix("frontend/dist", http.FileServer(http.FS(assets))))

	server := &http.Server{
		Addr:              "127.0.0.1:9732",
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           mux,
	}
	fmt.Printf("http://127.0.0.1:%v", port)
	browser.OpenURL("http://127.0.0.1:9732")
	log.Println(server.ListenAndServe())
}

var (
	port string
)

func init() {
	flag.StringVar(&port, "port", "9732", "port")
	flag.Parse()
}

func indexH(port, apiPort int) http.HandlerFunc {
	b, err := assets.ReadFile("frontend/dist/index.html")
	if err != nil {
		panic(err)
	}
	b = bytes.Replace(b, []byte(`<meta charset="UTF-8" />`), []byte("<meta charset=\"utf-8\" />\n"+
		"<script>window._player = {cors:\"http://127.0.0.1:"+strconv.Itoa(port)+"/\",api:\"http://127.0.0.1:"+strconv.Itoa(apiPort)+"/\"}</script>"), -1)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(b)
	}
}

func AddPrefix(prefix string, h http.Handler) http.Handler {
	if prefix == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			r.URL.Path = prefix
		} else {
			r.URL.Path = prefix + r.URL.Path
		}
		h.ServeHTTP(w, r)
	})
}
