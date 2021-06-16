package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type config struct {
	port int
	env  string
}

var version = "1.0.0"

type application struct {
	config config
	logger *log.Logger
}

func main() {
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.Parse()

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	app := &application{
		config: cfg,
		logger: logger,
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.port),
		Handler: app.routes(),
	}
	srv.IdleTimeout = time.Minute
	srv.ReadTimeout = 10 * time.Second
	srv.WriteTimeout = 30 * time.Second

	app.logger.Printf("starting %s server on %s", app.config.env,
		srv.Addr)

	err := srv.ListenAndServe()
	if err != nil {
		app.logger.Fatal(err)
	}
}
