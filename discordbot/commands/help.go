package commands

import (
	"context"
	"github.com/bwmarrin/discordgo"
	"strings"
)

func init() {
	Register(&Command{
		Name:        "help",
		Description: "Show this help message",
		Handler: func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
			var b strings.Builder
			b.WriteString("**Minecraft Server Bot Commands:**\n")
			for _, cmd := range All() {
				b.WriteString("?" + cmd.Name + " - " + cmd.Description + "\n")
			}
			embed := &discordgo.MessageEmbed{
				Title:       "Help",
				Description: b.String(),
				Color:       0x3498db,
			}
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		},
	})
}
