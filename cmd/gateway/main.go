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
	configFile = flag.String("config", "", "The config file (if not set will use the default configuration)")
)

func main() {
	flag.Parse()

	cfg := server.DefaultConfig()
	if *configFile != "" {
		f, err := os.Open(*configFile)
		fatalOnErr(err)
		err = yaml.NewDecoder(f).Decode(cfg)
		fatalOnErr(err)
	}

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
