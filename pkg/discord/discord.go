package discord

type AuthenticatedUser struct {
	DiscordID string
	Username  string
}

type UserRepository interface {
	GetDiscordUserByID(string) (*AuthenticatedUser, error)
}
