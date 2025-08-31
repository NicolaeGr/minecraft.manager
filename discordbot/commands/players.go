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
			} else {
				color = 0x00ff00
			}
			playerList := strings.Join(players, ", ")
			embed := &discordgo.MessageEmbed{
				Title:       "Players Online",
				Description: fmt.Sprintf("%d/%d: %s", count, max, playerList),
				Color:       color,
			}
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		},
	})
}
