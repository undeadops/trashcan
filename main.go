package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/undeadops/trashcan/pkg/server"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

var (
	serverPort  int
	metricsPort int
	s3Bucket    string
	clientCAs   string
	certFile    string
	keyFile     string
	debug       bool
)

func main() {
	flag.IntVar(&serverPort, "port", 3000, "Server Listening Port")
	flag.IntVar(&metricsPort, "metrics-port", 9090, "Metrics Listening Port")
	flag.StringVar(&s3Bucket, "bucket", "", "S3 Bucket to use")
	flag.BoolVar(&debug, "debug", false, "Debug Logging")
	flag.StringVar(&clientCAs, "client-ca", "", "File Paths to Client CA certificats, separated by commas")
	flag.StringVar(&certFile, "cert", "ssl/server.pem", "Server SSL Certificate")
	flag.StringVar(&keyFile, "key", "ssl/server-key.pem", "Server SSL Certificate Key")
	flag.Parse()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Caller().Logger()
	if debug {
		logger = zerolog.New(os.Stdout).Level(zerolog.DebugLevel).With().Timestamp().Caller().Logger()
	}

	s := server.SetupServer(ctx, logger, s3Bucket)
	router := s.Router()

	// Check if we have client certs to permit.
	tlsconfig := &tls.Config{}
	var err error
	if clientCAs != "" {
		tlsconfig, err = server.SetupClientCerts(clientCAs)
		if err != nil {
			logger.Error().Msgf("error setting client CA certs")
		}
	}

	svr := &http.Server{
		Addr:      fmt.Sprintf(":%d", serverPort),
		TLSConfig: tlsconfig,
		Handler:   router,
	}

	// Serve Metrics
	go func() {
		s.Logger.Info().Msgf("metrics listening at :%d", metricsPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), promhttp.Handler()); err != nil {
			s.Logger.Panic().Msgf("error serving metrics on :%d [%s]", metricsPort, err.Error())
		}
	}()

	go func() {
		s.Logger.Info().Msgf("api listening at :%d", serverPort)
		if err := svr.ListenAndServeTLS(certFile, keyFile); err != nil {
			s.Logger.Panic().Msgf("error serving api on :%d [%s]", serverPort, err.Error())
		}
	}()

	killSwitch := <-interrupt
	switch killSwitch {
	case os.Interrupt:
		s.Logger.Info().Msg("Got SIGINT, Shutting down server...")
	case syscall.SIGTERM:
		s.Logger.Info().Msg("Got SIGTERM, Shuttind down server...")
	}

	s.Logger.Info().Msg("server exiting....")
}
