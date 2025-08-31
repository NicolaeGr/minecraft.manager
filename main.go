package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"electrolit.biz/minecraft.manager/autostop"
	"electrolit.biz/minecraft.manager/manager"
	"electrolit.biz/minecraft.manager/webui"
)

func main() {
	workingPath := flag.String("workingPath", ".", "Path to use as working directory")
	flag.Parse()
	fmt.Println("App workingPath:", *workingPath)
	os.Chdir(*workingPath)

	mgr := manager.NewServerManager(*workingPath)

	go autostop.StartIdleWatcher(mgr)
	go webui.StartWebUI(mgr)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Terminating, stopping Minecraft server if running...")
		mgr.Stop()
		os.Exit(0)
	}()

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c // Block here until a signal is received
	fmt.Println("Terminating, stopping Minecraft server if running...")
	mgr.Stop()
	os.Exit(0)
}
