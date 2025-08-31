package commands

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func init() {
	Register(&Command{
		Name:        "ping",
		Description: "Test bot responsiveness",
		Handler: func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
			pong := "Pong!"
			if len(args) > 0 {
				pong = fmt.Sprintf("Pong! Args: %v", args)
			}
			s.ChannelMessageSend(m.ChannelID, pong)
		},
	})
}
