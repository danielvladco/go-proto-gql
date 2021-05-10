package gateway

import (
	"github.com/danielvladco/go-proto-gql/pkg/server"
	"github.com/rs/cors"
)

type Config struct {
	Grpc       *server.Grpc  `json:"grpc"`
	Cors       *cors.Options `json:"cors"`
	Playground *bool         `json:"playground"`
	Address    string        `json:"address"`
	Tls *server.Tls
}
