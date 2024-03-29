package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/registry/lib"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

func proxy(c *gin.Context) {
	remote, err := url.Parse(os.Getenv("REGISTRY_ADDRESS"))
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.Header.Get("Location") != "" {
			log.Println(resp.Header.Get("Location"))
			location, err := url.Parse(resp.Header.Get("Location"))
			if err != nil {
				log.Println(err)
				return nil
			}
			location.Scheme = "https"
			location.Host = os.Getenv("HOST")
			resp.Header.Set("Location", location.String())
		}
		return nil
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}

	db.Open()

	r := gin.Default()

	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(os.Getenv("HOST")),
		Cache:      autocert.DirCache("./.cache"),
		Client:     &acme.Client{DirectoryURL: os.Getenv("DIRECTORY_URL")},
	}

	if os.Getenv("REGISTRY_ADDRESS") != "" {
		r.NoRoute(proxy)
	}
	r.GET("/auth", lib.Handler(&m))
	s := &http.Server{
		Addr:      os.Getenv("ADDRESS"),
		TLSConfig: m.TLSConfig(),
		Handler:   r,
	}

	log.Println(s.ListenAndServeTLS("", ""))
}
