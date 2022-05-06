package main

import (
	"flag"
	"github.com/danielvladco/go-proto-gql/pkg/server"
	"log"
	"net"
	"net/http"
	"os"

	"gopkg.in/yaml.v2"
)

var (
	configFile = flag.String("cfg", "/opt/config.json", "")
)

func main() {
	flag.Parse()

	f, err := os.Open(*configFile)
	fatalOnErr(err)
	cfg := &server.Config{}
	err = yaml.NewDecoder(f).Decode(cfg)
	fatalOnErr(err)

	l, err := net.Listen("tcp", cfg.Address)
	fatalOnErr(err)
	log.Printf("[INFO] Gateway listening on address: %s\n", l.Addr())
	handler, err := server.Server(cfg)
	fatalOnErr(err)
	if cfg.Tls != nil {
		log.Fatal(http.ServeTLS(l, handler, cfg.Tls.Certificate, cfg.Tls.PrivateKey))
	}
	log.Fatal(http.Serve(l, handler))
}

func fatalOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
