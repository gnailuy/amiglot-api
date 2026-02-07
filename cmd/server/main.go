package main

import (
	"log"
	"net/http"

	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/gnailuy/amiglot-api/internal/db"
	httpx "github.com/gnailuy/amiglot-api/internal/http"
)

func main() {
	cfg := config.Load()

	conn, err := db.New(cfg)
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	if conn != nil {
		defer conn.Close()
		log.Printf("database connected")
	} else {
		log.Printf("DATABASE_URL not set; starting without database")
	}

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, httpx.Router()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
