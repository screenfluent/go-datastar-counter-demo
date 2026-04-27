package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/pressly/goose/v3"
	datastar "github.com/starfederation/datastar-go/datastar"

	"github.com/szymon/go-datastar-counter-demo/internal/counter"
	"github.com/szymon/go-datastar-counter-demo/internal/store"
	"github.com/szymon/go-datastar-counter-demo/views"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	counterStore, cleanup, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	hub := counter.NewHub(counterStore)
	defer hub.Close()

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())
	e.Static("/static", "static")

	e.GET("/", func(c *echo.Context) error {
		snapshot, err := hub.Snapshot(c.Request().Context())
		if err != nil {
			return err
		}
		var page bytes.Buffer
		if err := views.Page(snapshot).Render(c.Request().Context(), &page); err != nil {
			return err
		}
		return c.HTMLBlob(http.StatusOK, page.Bytes())
	})

	e.GET("/events", func(c *echo.Context) error {
		updates, unsubscribe, err := hub.Subscribe(c.Request().Context())
		if err != nil {
			return err
		}
		defer unsubscribe()

		sse := datastar.NewSSE(rawResponseWriter(c.Response()), c.Request())
		for {
			select {
			case snapshot := <-updates:
				if err := sse.PatchElementTempl(views.CounterCard(snapshot), datastar.WithSelectorID("counter-card")); err != nil {
					return err
				}
			case <-c.Request().Context().Done():
				return nil
			}
		}
	})

	e.POST("/counter/increment", changeCounter(hub, 1))
	e.POST("/counter/decrement", changeCounter(hub, -1))
	e.POST("/counter/reset", func(c *echo.Context) error {
		snapshot, err := hub.Reset(c.Request().Context())
		if err != nil {
			return err
		}
		return datastar.NewSSE(rawResponseWriter(c.Response()), c.Request()).
			PatchElementTempl(views.CounterCard(snapshot), datastar.WithSelectorID("counter-card"))
	})

	serverErrors := make(chan error, 1)
	go func() {
		addr := ":" + env("PORT", "8080")
		slog.Info("listening", "addr", addr)
		serverErrors <- (&echo.StartConfig{
			Address:         addr,
			GracefulTimeout: 10 * time.Second,
		}).Start(ctx, e)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func changeCounter(hub *counter.Hub, delta int) echo.HandlerFunc {
	return func(c *echo.Context) error {
		snapshot, err := hub.Change(c.Request().Context(), delta)
		if err != nil {
			return datastar.NewSSE(rawResponseWriter(c.Response()), c.Request()).
				PatchElementTempl(views.CounterCard(snapshot), datastar.WithSelectorID("counter-card"))
		}
		return datastar.NewSSE(rawResponseWriter(c.Response()), c.Request()).
			PatchElementTempl(views.CounterCard(snapshot), datastar.WithSelectorID("counter-card"))
	}
}

func rawResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	type unwrapper interface {
		Unwrap() http.ResponseWriter
	}
	if wrapped, ok := w.(unwrapper); ok {
		return wrapped.Unwrap()
	}
	return w
}

func openStore(ctx context.Context) (counter.Store, func(), error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, nil, errors.New("DATABASE_URL is required")
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, nil, err
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	if err := runMigrations(sqlDB); err != nil {
		pool.Close()
		return nil, nil, err
	}

	slog.Info("connected to postgres")
	return store.NewPostgres(pool), pool.Close, nil
}

func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(os.DirFS("."))
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
