package main

import (
	"log"
	"net/http"
	"time"

	"itstudio/server"
)

func main() {
	cfg := server.LoadConfig()
	db, err := server.OpenDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	app := server.New(db, cfg)
	srv := &http.Server{Addr: cfg.Addr, Handler: app.Router(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second}
	log.Printf("IT Studio listening on %s", cfg.Addr)
	server.LogServer(srv.ListenAndServe())
}
