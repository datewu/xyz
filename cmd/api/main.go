package main

import (
	"context"
	"database/sql"
	"flag"
	"strings"
	"time"

	"github.com/datewu/gtea"
	"github.com/datewu/xyz/internal/data"
	_ "github.com/lib/pq"
)

var models data.Models

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
	metrics bool
}

var (
	version   = "1.0.0"
	buildTime string
)

func main() {
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "postgreSQL dsn")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "postgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", false, "Enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "a3e7d95345e9ef", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "202ec0eaa7e43d", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender")

	// -cors-trusted-origins="http://localhost:9000 http://localhost:9001"
	flag.Func("cors-trusted-origins", "Tursted CORS origins (space separated)", func(v string) error {
		cfg.cors.trustedOrigins = strings.Fields(v)
		return nil
	})

	flag.BoolVar(&cfg.metrics, "metrics", false, "Enable expvar metrics")

	flag.Parse()

	cnf := &gtea.Config{
		Port:    cfg.port,
		Env:     cfg.env,
		Metrics: cfg.metrics,
	}

	app := gtea.NewApp(cnf)

	app.Logger.PrintInfo("build info", map[string]string{
		"version":   version,
		"buildTime": buildTime,
	})

	db, err := openDB(cfg)
	if err != nil {
		app.Logger.PrintFatal(err, nil)
	}
	models = data.NewModels(db)
	app.SetDB(db)
	app.Logger.PrintInfo("database connection pool established", nil)

	routes := setRoutes()
	err = app.Serve(routes)
	if err != nil {
		app.Logger.PrintFatal(err, nil)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}
