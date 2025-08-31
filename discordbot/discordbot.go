package discordbot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"electrolit.biz/minecraft.manager/discordbot/commands"
	"electrolit.biz/minecraft.manager/manager"

	"github.com/bwmarrin/discordgo"
)

func StartBot(mgr *manager.ServerManager) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		fmt.Println("DISCORD_BOT_TOKEN not set")
		return
	}
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	_ = commands.All()

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}

		content := m.Content
		if len(content) > 0 && content[0] == '?' {
			parts := strings.Fields(content[1:])
			if len(parts) == 0 {
				return
			}
			cmd := commands.Get(parts[0])
			if cmd != nil {
				cmd.Handler(context.Background(), s, m, parts[1:])
				return
			}
		}
	})

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	if err := dg.Open(); err != nil {
		fmt.Println("Error opening Discord connection:", err)
		return
	}

	fmt.Println("Discord bot is now running")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	dg.Close()
}
