package database

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/dallasurbanists/events-sync/pkg/discord"
	"github.com/dallasurbanists/events-sync/pkg/event"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Store struct {
	*sqlx.DB
	Events                    event.Repository
	AuthenticatedDiscordUsers discord.UserRepository
}

type DB struct {
	*sqlx.DB
}

// Connect establishes a database connection
func Connect(connStr string) (*Store, error) {
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return &Store{
		db,
		&EventRepository{db},
		&AuthenticatedDiscordUserRepository{db},
	}, nil
}

var columnsCache sync.Map // map[reflect.Type]string

func DBColumns[T any]() string {
	var zero T
	rt := reflect.TypeOf(zero)
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if s, ok := columnsCache.Load(rt); ok {
		return s.(string)
	}
	cols := collectDBTags(rt)
	csv := strings.Join(cols, ",")
	columnsCache.Store(rt, csv)
	return csv
}

func collectDBTags(rt reflect.Type) []string {
	if rt.Kind() != reflect.Struct {
		return nil
	}
	var out []string

	var walk func(reflect.Type)
	walk = func(t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			// Skip unexported non-embedded fields.
			if f.PkgPath != "" && !f.Anonymous {
				continue
			}

			// Recurse into embedded structs (T or *T).
			if f.Anonymous {
				ft := f.Type
				if ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}
				if ft.Kind() == reflect.Struct {
					walk(ft)
					continue
				}
			}

			tag := f.Tag.Get("db")
			if tag == "" || tag == "-" {
				continue
			}
			// Handle options like `db:"name,omitempty"`
			if idx := strings.IndexByte(tag, ','); idx >= 0 {
				tag = tag[:idx]
			}
			out = append(out, tag)
		}
	}
	walk(rt)
	return out
}
