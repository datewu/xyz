package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/datewu/xyz/internal/data"
	"github.com/datewu/xyz/internal/jsonlog"
	"github.com/datewu/xyz/internal/mailer"
	_ "github.com/lib/pq"
)

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

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

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

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)
	logger.PrintInfo("build info", map[string]string{
		"version":   version,
		"buildTime": buildTime,
	})

	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	if cfg.metrics {
		expvar.NewString("version").Set(version)
		expvar.Publish("goroutines", expvar.Func(func() interface{} {
			return runtime.NumGoroutine()
		}))
		expvar.Publish("database", expvar.Func(func() interface{} {
			return db.Stats()
		}))
		expvar.Publish("timestamp", expvar.Func(func() interface{} {
			return time.Now().Unix()
		}))
	}
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host,
			cfg.smtp.port, cfg.smtp.username, cfg.smtp.password,
			cfg.smtp.sender),
	}

	err = app.serve()
	if err != nil {
		app.logger.PrintFatal(err, nil)
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
