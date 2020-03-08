package main

import (
	"flag"

	"github.com/wayjam/github-release-proxy/server"
)

func main() {
	port := flag.Int("p", 80, "listens on the specify port")

	flag.Parse()

	srv := server.New()
	srv.Start(*port)
}
