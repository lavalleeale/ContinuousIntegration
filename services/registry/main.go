package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/registry/lib"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

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

	r.GET("/auth", lib.Handler(&m))
	s := &http.Server{
		Addr:      os.Getenv("ADDRESS"),
		TLSConfig: m.TLSConfig(),
		Handler:   r,
	}

	s.ListenAndServeTLS("", "")
}
