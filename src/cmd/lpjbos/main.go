package main

import (
	"log"
	"os"

	"lpjbos/internal/app"
)

func main() {
	if err := app.Init(); err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		addr = ":" + port
	}
	log.Fatal(app.Run(addr))
}
