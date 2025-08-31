package commands

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type Command struct {
	Name        string
	Description string
	Handler     func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string)
}

var registry = map[string]*Command{}

func Register(cmd *Command) {
	registry[cmd.Name] = cmd
}

func Get(name string) *Command {
	return registry[name]
}

func All() []*Command {
	cmds := make([]*Command, 0, len(registry))
	for _, c := range registry {
		cmds = append(cmds, c)
	}
	return cmds
}
