package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/middleware"
	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"

	"github.com/undeadops/trashcan/pkg/metrics"
)

type Server struct {
	Logger zerolog.Logger
	Client *s3.Client
	Bucket string
}

func SetupServer(ctx context.Context, logger zerolog.Logger, bucket string) *Server {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error().Msgf("error loading default AWS config: %s", err.Error())
	}

	client := s3.NewFromConfig(cfg)

	return &Server{
		Logger: logger,
		Client: client,
		Bucket: bucket,
	}
}

func (s *Server) Router() *chi.Mux {
	prom := metrics.NewPrometheusMiddleware()

	r := chi.NewRouter()

	r.Use(middleware.Heartbeat("/ping"))
	r.Use(httplog.RequestLogger(s.Logger))
	r.Use(prom.Handler)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", s.index)
	r.Get("/hello", s.hello)

	return r
}

func SetupClientCerts(certs string) (*tls.Config, error) {
	clientCerts := strings.Split(certs, ",")
	caCertPool := x509.NewCertPool()

	for _, c := range clientCerts {
		caCert, err := ioutil.ReadFile(c)
		if err != nil {
			fmt.Printf("failed to read ca pem certificate %s: %v", c, err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}

	tlsConfig := &tls.Config{
		ClientCAs: caCertPool,
		//ClientAuth: tls.RequireAndVerifyClientCert,
		ClientAuth: tls.VerifyClientCertIfGiven,
	}
	tlsConfig.BuildNameToCertificate()

	return tlsConfig, nil
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJSON(w, code, map[string]string{"message": msg})
}
