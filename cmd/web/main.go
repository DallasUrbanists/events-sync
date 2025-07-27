package main

import (
	"flag"
	"log"
	"os"

	"github.com/dallasurbanists/events-sync/internal/database"
	"github.com/dallasurbanists/events-sync/internal/server"
)

func main() {
	var (
		port           = flag.String("port", "8080", "Port to run the server on")
		dbURL          = flag.String("database", "", "Database connection URL")
	)
	flag.Parse()

	if *dbURL == "" {
		*dbURL = os.Getenv("DATABASE_URL")
		if *dbURL == "" {
			log.Fatalf("No DATABASE_URL given")
		}
	}

	db, err := database.Connect(*dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	log.Fatal(server.NewServer(db, server.NewAppOpts{Port: *port}).Server.ListenAndServe())
}
