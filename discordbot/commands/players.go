package commands

import (
	"context"
	"fmt"
	"strings"

	"electrolit.biz/minecraft.manager/manager"
	"github.com/bwmarrin/discordgo"
)

func init() {
	Register(&Command{
		Name:        "players",
		Description: "Show online player count and names",
		Handler: func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
			mgr := ctx.Value("mgr").(*manager.ServerManager)
			state, _ := mgr.Status()
			if state != manager.StateRunning {
				embed := &discordgo.MessageEmbed{
					Title:       "Server Offline",
					Description: "The Minecraft server is currently offline.",
					Color:       0xff0000,
				}
				s.ChannelMessageSendEmbed(m.ChannelID, embed)
				return
			}
			count, max, players, err := mgr.GetPlayerList()
			if err != nil {
				embed := &discordgo.MessageEmbed{
					Title:       "Players Online",
					Description: "Error retrieving player list: " + err.Error(),
					Color:       0xff0000,
				}
				s.ChannelMessageSendEmbed(m.ChannelID, embed)
				return
			}
			var color int
			if count == 0 {
				color = 0xff0000
			} else if max > 0 && count < max/2 {
				color = 0xffff00
			} else {
				color = 0x00ff00
			}
			msg := fmt.Sprintf("Players online: %d/%d\n%s", count, max, strings.Join(players, ", "))
			embed := &discordgo.MessageEmbed{
				Title:       "Players Online",
				Description: msg,
				Color:       color,
			}
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		},
	})
}
