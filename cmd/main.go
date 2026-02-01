package main

import (
	"log/slog"
	"os"

	server "github.com/tender-barbarian/gniot/server"
)

func main() {
	err := server.Run()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
