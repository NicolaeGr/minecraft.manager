package commands

import (
	"context"

	"electrolit.biz/minecraft.manager/manager"
	"github.com/bwmarrin/discordgo"
)

func init() {
	Register(&Command{
		Name:        "status",
		Description: "Show Minecraft server status",
		Handler: func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
			mgr := ctx.Value("mgr").(*manager.ServerManager)
			state, _ := mgr.Status()
			color := 0xff0000
			switch state {
			case manager.StateRunning:
				color = 0x00ff00
			case manager.StateStarting:
				color = 0xffff00
			}
			embed := &discordgo.MessageEmbed{
				Title:       "Server Status",
				Description: "Status: " + string(state),
				Color:       color,
			}
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		},
	})
}
