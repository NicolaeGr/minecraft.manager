package commands

import (
	"context"

	"electrolit.biz/minecraft.manager/autostop"
	"github.com/bwmarrin/discordgo"
)

func init() {
	Register(&Command{
		Name:        "countdown",
		Description: "Show idle countdown timer",
		Handler: func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
			response := autostop.GetRemainingTime()
			embed := &discordgo.MessageEmbed{
				Title:       "Idle Countdown",
				Description: response,
				Color:       0x3498db,
			}
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		},
	})
}
