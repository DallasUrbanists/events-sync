package server

import (
	"fmt"
	"net/http"

	"github.com/dallasurbanists/events-sync/internal/database"
)

type Server struct {
	db *database.DB
	host string
	port string
	Server http.Server
}

type NewAppOpts struct {
	Host string
	Port string
}

func NewServer (db *database.DB, o NewAppOpts) *Server {
	s := &Server{db: db}

	addr := o.Host
	if o.Port != "" {
		addr += fmt.Sprintf(":%v", o.Port)
	}

	s.Server = http.Server{
		Addr: addr,
		Handler: s.newConfiguredRouter(),
	}

	return s
}
