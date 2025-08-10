package discordbot

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"electrolit.biz/minecraft.manager/manager"

	"github.com/mcstatus-io/mcutil/v4/query"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func getFallbackStatus() (count int, max int, playerNames []string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	response, err := query.Full(ctx, "localhost", 25565)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("timeout waiting for player list: %w", err)
	}
	max = 0
	count = 0
	if v, ok := response.Data["maxplayers"]; ok {
		fmt.Sscanf(v, "%d", &max)
	}
	if v, ok := response.Data["numplayers"]; ok {
		fmt.Sscanf(v, "%d", &count)
	}

	return count, max, response.Players, nil
}

func StartBot(mgr *manager.ServerManager) {
	godotenv.Load()
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

	isStarting := false

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}

		switch m.Content {
		case "!test":
			fmt.Println("DiscordBot: !test command received")
			count, max, names, err := mgr.GetPlayerList()
			fmt.Printf("DiscordBot: GetPlayerList returned count=%d, max=%d, names=%v, err=%v\n", count, max, names, err)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Error getting player list: "+err.Error())
				fmt.Println("DiscordBot: Error getting player list:", err)
				return
			}
			msg := fmt.Sprintf("Players online: %d/%d\n%s", count, max, strings.Join(names, ", "))
			s.ChannelMessageSend(m.ChannelID, msg)

		case "!ping":
			colors := []int{0xff0000, 0xffa500, 0xffff00, 0x00ff00, 0x0000ff, 0xff00ff}
			color := colors[rand.Intn(len(colors))]
			pong := "Pong!"
			if rand.Intn(2) == 0 {
				pong = "💣"
			}
			embed := &discordgo.MessageEmbed{
				Title:       "Ping",
				Description: pong,
				Color:       color,
			}
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		case "!status":
			status := mgr.Status()
			color := 0xff0000
			switch status {
			case "running":
				color = 0x00ff00
			case "starting":
				color = 0xffff00
			}
			embed := &discordgo.MessageEmbed{
				Title:       "Server Status",
				Description: "Status: " + status,
				Color:       color,
			}
			s.ChannelMessageSendEmbed(m.ChannelID, embed)

		case "!players":
			if mgr.Status() != "running" {
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

				count, max, players, err = getFallbackStatus()
				if err != nil {
					embed := &discordgo.MessageEmbed{
						Title:       "Players Online",
						Description: "Error retrieving player list: " + err.Error(),
						Color:       0xff0000,
					}
					s.ChannelMessageSendEmbed(m.ChannelID, embed)
					return
				}
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
		case "!start":
			if isStarting {
				embed := &discordgo.MessageEmbed{
					Title:       "Don't rush me",
					Description: "The server is still starting...",
					Color:       0xffa500,
				}
				msg, _ := s.ChannelMessageSendEmbed(m.ChannelID, embed)
				go func() {
					time.Sleep(2 * time.Second)
					s.ChannelMessageDelete(m.ChannelID, msg.ID)
				}()
				return
			}
			if mgr.Status() == "running" {
				embed := &discordgo.MessageEmbed{
					Title:       "Server Start",
					Description: "Server is already running.",
					Color:       0x00ff00,
				}
				s.ChannelMessageSendEmbed(m.ChannelID, embed)
				return
			}
			isStarting = true
			embed := &discordgo.MessageEmbed{
				Title:       "Starting Minecraft Server...",
				Description: "Please wait, this may take a few minutes.",
				Color:       0xffa500, // orange
			}
			msg, _ := s.ChannelMessageSendEmbed(m.ChannelID, embed)
			go func() {
				err := mgr.Start()
				if err != nil {
					isStarting = false
					s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, &discordgo.MessageEmbed{
						Title:       "Error starting server",
						Description: err.Error(),
						Color:       0xff0000,
					})
					return
				}
				for i := 0; i < 240; i++ {
					if mgr.Status() == "running" {
						isStarting = false
						s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, &discordgo.MessageEmbed{
							Title:       "Server Started!",
							Description: "Minecraft server is now online.",
							Color:       0x00ff00,
						})
						return
					}
					time.Sleep(1 * time.Second)
				}
				isStarting = false
				s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, &discordgo.MessageEmbed{
					Title:       "Server start timed out",
					Description: "Minecraft server did not come online in time.",
					Color:       0xff0000,
				})
			}()
		case "!help":
			helpMsg := "**Minecraft Server Bot Commands:**\n" +
				"`!help` - Show this help message\n" +
				"`!ping` - Test bot responsiveness\n" +
				"`!status` - Show Minecraft server status\n" +
				"`!start` - Start the Minecraft server\n" +
				"`!players` - Show online player count and names"
			embed := &discordgo.MessageEmbed{
				Title:       "Help",
				Description: helpMsg,
				Color:       0x3498db,
			}
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
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
