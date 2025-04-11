package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	version string
	commit  string
	date    string

	endpoints resourceFlags
	interval  int
	logLevel  string
)

type resourceFlags []*Resource

// String is an implementation of the flag.Value interface
func (f *resourceFlags) String() string {
	return fmt.Sprintf("%v", *f)
}

// Set is an implementation of the flag.Value interface
func (f *resourceFlags) Set(value string) error {
	var err error

	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return fmt.Errorf("invalid endpoint")
	}

	status := http.StatusOK
	if len(parts) > 1 && parts[1] != "" {
		status, err = strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid status code: %w", err)
		}
	}

	timeout := 5 * time.Second
	if len(parts) > 2 && parts[2] != "" {
		timeout, err = time.ParseDuration(parts[2])
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
	}

	*f = append(*f, NewResource(parts[0], status, timeout))
	return nil
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}

	slog.Warn("invalid log level", "level", level, "fallback", "info")
	return slog.LevelInfo
}

func parseFlags() {
	printVersion := flag.Bool("version", false, "print version and exit")
	flag.Var(&endpoints, "endpoint", "endpoints to monitor (endpoint[,status[,timeout]])")
	flag.IntVar(&interval, "interval", 60, "interval (in seconds) between pings")
	flag.StringVar(&logLevel, "log-level", "info", "log level")
	flag.Parse()

	if *printVersion {
		fmt.Printf("simplemonitor version %s, commit %s (%s)\n", version, commit[:7], date)
		os.Exit(0)
	}

	slog.SetLogLoggerLevel(parseLogLevel(logLevel))

	if len(endpoints) == 0 {
		slog.Error("no endpoints provided")
		os.Exit(1)
	}

	if interval <= 0 {
		slog.Error("interval must be greater than 0")
		os.Exit(1)
	}
}

func main() {
	parseFlags()

	var tasks []Task
	for _, endpoint := range endpoints {
		tasks = append(tasks, func() error {
			return endpoint.Ping(context.Background())
		})
	}

	pool := NewWorkerPool(4)
	pool.StartScheduler(*time.NewTicker(time.Duration(interval) * time.Second), tasks...)
	pool.Start()

	// Handle graceful shutdown
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signalChannel
		switch sig {
		case os.Interrupt:
			pool.Shutdown(5 * time.Second)
		case syscall.SIGTERM:
			pool.Shutdown(0)
		}
	}()
}
