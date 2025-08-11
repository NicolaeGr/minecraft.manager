package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"electrolit.biz/minecraft.manager/autostop"
	"electrolit.biz/minecraft.manager/discordbot"
	"electrolit.biz/minecraft.manager/manager"
)

func main() {
	workingPath := flag.String("workingPath", ".", "Path to use as working directory")
	flag.Parse()
	fmt.Println("App workingPath:", *workingPath)
	os.Chdir(*workingPath)

	mgr := manager.NewServerManager()
	fmt.Println("Starting Discord bot...")

	// Start idle watcher in background
	go autostop.StartIdleWatcher(mgr)

	// Handle termination signals to stop the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Terminating, stopping Minecraft server if running...")
		mgr.Stop()
		os.Exit(0)
	}()

	discordbot.StartBot(mgr)
	// The server will be started/stopped via Discord bot commands
}
