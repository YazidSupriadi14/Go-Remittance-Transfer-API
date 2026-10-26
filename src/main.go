// Command finshot runs the Remittance Transfer API.
//
// main is the composition root: it is the only place that knows about every layer
// and wires the concrete implementations together.
//
//	httpapi (HTTP in/out) -> transfer (business rules) -> storage (SQLite)
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"finshot/src/httpapi"
	"finshot/src/storage"
	"finshot/src/transfer"
)

func main() {
	// Every setting can come from a flag or an environment variable; the flag wins.
	port := flag.String("port", envOrDefault("PORT", "8080"), "HTTP port")
	schemaPath := flag.String("schema", envOrDefault("SCHEMA_PATH", "schema.sql"), "path to schema.sql")
	dbPath := flag.String("db", os.Getenv("DB_PATH"), "SQLite file to create on start (empty = in-memory)")
	flag.Parse()

	db, err := storage.Open(*dbPath, *schemaPath)
	if err != nil {
		log.Fatalf("start database: %v", err)
	}
	defer db.Close()

	if *dbPath == "" {
		log.Printf("database: in-memory (data is lost when the app stops)")
	} else {
		absolute, _ := filepath.Abs(*dbPath)
		log.Printf("database: file %s (recreated on every start)", absolute)
	}

	service := transfer.NewService(
		storage.NewTransferRepository(db),
		storage.NewCustomerRepository(db),
		time.Now,
		uuid.NewString,
	)
	router := httpapi.NewRouter(httpapi.NewTransferHandler(service))

	addr := ":" + *port
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Finshot API listening on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
