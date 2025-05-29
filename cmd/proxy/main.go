package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime"

	"github.com/foxinuni/distribuidos-proxy/internal/handler"
	"github.com/foxinuni/distribuidos-proxy/internal/services"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var config Config

func init() {
	// Load config from flags
	flag.IntVar(&config.Port, "port", 4444, "Port to listen on")
	flag.IntVar(&config.Workers, "workers", runtime.NumCPU(), "Number of worker goroutines")
	flag.BoolVar(&config.Debug, "debug", false, "Enable debug logging")
	flag.Parse()

	// Set up zerolog logger for debug and pretty print
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if config.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func main() {
	// 1. Construct services for server
	serializerService := services.NewJsonModelSerializer()

	// 2. Boostrap the proxy
	proxy := handler.NewServer(
		serializerService,

		// Optional server options
		handler.WithPort(config.Port),
		handler.WithWorkerCount(config.Workers),
		handler.WithServers(
			"tcp://178.156.158.116:5556",
			"tcp://192.168.0.105:5556",
		),
	)

	// 5. Start the server
	if err := proxy.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start server")
		os.Exit(1)
	}
	defer proxy.Stop()

	// 6. Wait for shutdown signal (CTRL+C)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}
