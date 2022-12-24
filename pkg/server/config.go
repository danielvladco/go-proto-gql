package server

import (
	"github.com/rs/cors"
)

type Tls struct {
	Certificate string `json:"certificate" yaml:"certificate"`
	PrivateKey  string `json:"private_key" yaml:"private_key"`
}

type Authentication struct {
	Tls *Tls `json:"tls" yaml:"tls"`
}

type Service struct {
	Address        string          `json:"address" yaml:"address"`
	Authentication *Authentication `json:"authentication" yaml:"authentication"`
	Reflection     bool            `json:"reflection" yaml:"reflection"`
	ProtoFiles     []string        `json:"proto_files" yaml:"proto_files"`
}
type Config struct {
	Grpc       *Grpc         `json:"grpc" yaml:"grpc"`
	Cors       *cors.Options `json:"cors" yaml:"cors"`
	Playground *bool         `json:"playground" yaml:"playground"`
	Address    string        `json:"address" yaml:"address"`
	Tls        *Tls          `json:"tls" yaml:"tls"`
}

func DefaultConfig() *Config {
	return &Config{
		Address:    ":8080",
		Cors:       &cors.Options{},
		Grpc:       &Grpc{},
		Playground: &[]bool{true}[0],
		Tls:        nil,
	}
}
