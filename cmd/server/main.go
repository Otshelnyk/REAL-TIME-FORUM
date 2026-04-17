package main

import (
	"log"
	"os"
	"strings"

	"github.com/ndanbaev/forum/internal/server"
)

func main() {
	addr := strings.TrimSpace(os.Getenv("ADDR"))
	if addr == "" {
		if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
			addr = ":" + port
		}
	}

	dbPath := strings.TrimSpace(os.Getenv("SQLITE_PATH"))

	if err := server.Run(server.Config{Addr: addr, SQLitePath: dbPath}); err != nil {
		log.Fatal(err)
	}
}

