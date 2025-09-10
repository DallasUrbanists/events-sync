package server

import (
	"fmt"
	"net/http"

	"github.com/dallasurbanists/events-sync/internal/config"
	"github.com/dallasurbanists/events-sync/internal/database"
)

type Server struct {
	db           *database.Store
	config       *config.Config
	discordConfig *config.DiscordConfig
	jwtConfig    *config.JWTConfig
	host         string
	port         string
	Server       http.Server
}

type NewAppOpts struct {
	Host   string
	Port   string
	Config *config.Config
}

func NewServer(db *database.Store, o NewAppOpts) (*Server, error) {
	// Load Discord configuration from environment variables
	discordConfig, err := config.LoadDiscordConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Discord config: %v", err)
	}

	// Load JWT configuration from environment variables
	jwtConfig, err := config.LoadJWTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load JWT config: %v", err)
	}

	s := &Server{
		db:            db,
		config:        o.Config,
		discordConfig: discordConfig,
		jwtConfig:     jwtConfig,
	}

	addr := o.Host
	if o.Port != "" {
		addr += fmt.Sprintf(":%v", o.Port)
	}

	s.Server = http.Server{
		Addr:    addr,
		Handler: s.newConfiguredRouter(),
	}

	return s, nil
}
