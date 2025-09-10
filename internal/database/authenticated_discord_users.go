package database

import (
	"fmt"
	"time"

	"github.com/dallasurbanists/events-sync/pkg/discord"
	"github.com/jmoiron/sqlx"
)

type AuthenticatedDiscordUserRepository struct {
	*sqlx.DB
}

type AuthenticatedDiscordUser struct {
	ID        int       `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	DiscordID string `db:"discord_id"`
	Username  string `db:"username"`
}

func (db *AuthenticatedDiscordUserRepository) GetDiscordUserByID(discordID string) (*discord.AuthenticatedUser, error) {
	var user AuthenticatedDiscordUser
	err := db.Get(&user, "SELECT * FROM authenticated_discord_users WHERE discord_id = $1", discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Discord user: %v", err)
	}

	return &discord.AuthenticatedUser{
		DiscordID: user.DiscordID,
		Username:  user.Username,
	}, nil
}
