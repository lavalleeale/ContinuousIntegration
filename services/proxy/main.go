package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}
	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClient(opt)

	mainHost := os.Getenv("MAIN_HOST")

	proxyHost := os.Getenv("PROXY_HOST")

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			val, err := rdb.Get(context.TODO(), strings.Split(r.In.Host, ".")[0]).Result()
			if err != nil {
				r.Out.URL.Scheme = "http"
				r.Out.URL.Host = mainHost
				r.Out.URL.Path = "/unknownProxy"
				return
			}
			r.Out.URL.Scheme = "http"
			r.Out.URL.Host = fmt.Sprintf("%s:%s", proxyHost, val)
			r.Out.Host = r.In.Host
		},
	}
	s := &http.Server{
		Addr:    os.Getenv("ADDRESS"),
		Handler: proxy,
	}
	err = s.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
