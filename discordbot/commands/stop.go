package commands

import (
	"context"
	"fmt"

	"electrolit.biz/minecraft.manager/discordbot"
	"electrolit.biz/minecraft.manager/manager"
	"github.com/bwmarrin/discordgo"
)

func init() {
	Register(&Command{
		Name:        "stop",
		Description: "Stop the Minecraft server",
		Handler: func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
			mgr := ctx.Value("mgr").(*manager.ServerManager)
			state, _ := mgr.Status()
			if state != manager.StateRunning {
				embed := &discordgo.MessageEmbed{
					Title:       "Server Stop",
					Description: "The Minecraft server is not running.",
					Color:       0xff0000,
				}
				s.ChannelMessageSendEmbed(m.ChannelID, embed)
				return
			}
			// Only allow stop if no players or admin
			count, _, _, err := mgr.GetPlayerList()
			if err == nil && count > 0 && !discordbot.IsAdmin(m.Author.ID) {
				embed := &discordgo.MessageEmbed{
					Title:       "Cannot Stop Server",
					Description: fmt.Sprintf("There are currently %d players online. Please ask them to leave before stopping the server.", count),
					Color:       0xff0000,
				}
				s.ChannelMessageSendEmbed(m.ChannelID, embed)
				return
			}
			err = mgr.Stop()
			if err != nil {
				embed := &discordgo.MessageEmbed{
					Title:       "Error stopping server",
					Description: err.Error(),
					Color:       0xff0000,
				}
				s.ChannelMessageSendEmbed(m.ChannelID, embed)
				return
			}
			embed := &discordgo.MessageEmbed{
				Title:       "Server Stopped",
				Description: "Minecraft server has been stopped.",
				Color:       0xff0000,
			}
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		},
	})
}
