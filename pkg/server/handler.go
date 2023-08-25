package server

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (s *Server) s3(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Duration(time.Second))
	defer cancel()

	output, err := s.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.Bucket),
	})
	if err != nil {
		s.Logger.Error().Msgf("error listing objects: %s", err.Error())
	}
	var items = make(map[string]int64)
	for _, object := range output.Contents {
		items[aws.ToString(object.Key)] = object.Size
	}
	respondWithJSON(w, http.StatusOK, items)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	clientCAs := r.TLS.PeerCertificates
	if len(clientCAs) == 0 {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	respondWithJSON(w, http.StatusOK, "OK!")
}

func (s *Server) hello(w http.ResponseWriter, r *http.Request) {
	clientCAs := r.TLS.PeerCertificates
	if len(clientCAs) == 0 {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	respondWithJSON(w, http.StatusOK, "HELLO!")
}
