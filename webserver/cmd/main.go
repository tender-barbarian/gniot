package main

import (
	"log"

	"github.com/tender-barbarian/gniot/webserver/internal/server"
)

func main() {
	err := server.Run()
	if err != nil {
		log.Fatal(err)
	}
}
