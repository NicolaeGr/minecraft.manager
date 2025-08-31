package commands

import (
	"context"
	"time"

	"electrolit.biz/minecraft.manager/manager"
	"github.com/bwmarrin/discordgo"
)

func init() {
	Register(&Command{
		Name:        "start",
		Description: "Start the Minecraft server",
		Handler: func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
			mgr := ctx.Value("mgr").(*manager.ServerManager)
			state, _ := mgr.Status()
			if state == manager.StateRunning {
				embed := &discordgo.MessageEmbed{
					Title:       "Server Start",
					Description: "Server is already running.",
					Color:       0x00ff00,
				}
				s.ChannelMessageSendEmbed(m.ChannelID, embed)
				return
			}
			embed := &discordgo.MessageEmbed{
				Title:       "Starting Minecraft Server...",
				Description: "Please wait, this may take a few minutes.",
				Color:       0xffa500,
			}
			msg, _ := s.ChannelMessageSendEmbed(m.ChannelID, embed)
			go func() {
				err := mgr.Start()
				if err != nil {
					s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, &discordgo.MessageEmbed{
						Title:       "Error starting server",
						Description: err.Error(),
						Color:       0xff0000,
					})
					return
				}
				for i := 0; i < 240; i++ {
					state, _ := mgr.Status()
					if state == manager.StateRunning {
						s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, &discordgo.MessageEmbed{
							Title:       "Server Started!",
							Description: "Minecraft server is now online.",
							Color:       0x00ff00,
						})
						return
					}
					time.Sleep(1 * time.Second)
				}
				s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, &discordgo.MessageEmbed{
					Title:       "Server start timed out",
					Description: "Minecraft server did not come online in time.",
					Color:       0xff0000,
				})
			}()
		},
	})
}
