package discord

import "time"

// InteractionRequest es la estructura enviada por Discord
type InteractionRequest struct {
	Version int `json:"version"`
	Type    int `json:"type"` // 1 = PING, 2 = APPLICATION_COMMAND, 3 = MESSAGE_COMPONENT
	Data    struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Type     int    `json:"type"`
		Options  []struct {
			Name  string `json:"name"`
			Type  int    `json:"type"`
			Value string `json:"value"`
		} `json:"options"`
		Attachments []struct {
			ID       string `json:"id"`
			Filename string `json:"filename"`
			Size     int    `json:"size"`
			URL      string `json:"url"`
		} `json:"attachments"`
	} `json:"data"`
	GuildID  string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
	Member   struct {
		User struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"user"`
	} `json:"member"`
	User struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
	ID    string `json:"id"`
	Token string `json:"token"`
	CreatedTimestamp time.Time `json:"created_timestamp"`
}

// ParseInteractionData extrae el comando y argumentos
func (ir *InteractionRequest) ParseCommand() (command string, args map[string]string) {
	command = ir.Data.Name
	args = make(map[string]string)
	for _, opt := range ir.Data.Options {
		args[opt.Name] = opt.Value
	}
	return
}

// UserID retorna el ID del usuario que ejecutó el comando
func (ir *InteractionRequest) UserID() string {
	// Intenta primero el member.user.id (en servidores)
	if ir.Member.User.ID != "" {
		return ir.Member.User.ID
	}
	// Fallback a user.id (en DMs o contextos sin member)
	return ir.User.ID
}
