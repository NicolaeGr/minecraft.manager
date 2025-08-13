package autostop

import (
	"fmt"
	"sync"
	"time"

	"electrolit.biz/minecraft.manager/manager"
)

var (
	lastOnlineMutex sync.Mutex
	lastOnlineTime  time.Time
)

func ResetIdleWatcher() {
	lastOnlineMutex.Lock()
	defer lastOnlineMutex.Unlock()
	lastOnlineTime = time.Now()
}

func CheckIdleAndStop(mgr *manager.ServerManager) {
	lastOnlineMutex.Lock()
	defer lastOnlineMutex.Unlock()

	status := mgr.Status()
	if status != "running" {
		lastOnlineTime = time.Time{} // Reset when server stops
		return
	}

	count, _, _, err := mgr.GetPlayerList()
	if err != nil {
		fmt.Println("IdleWatcher: error getting player list:", err)
		return
	}

	if count > 0 {
		if lastOnlineTime.IsZero() || time.Since(lastOnlineTime) > 1*time.Minute {
			lastOnlineTime = time.Now() // Reset when someone rejoins
		}
		return
	}

	if lastOnlineTime.IsZero() {
		lastOnlineTime = time.Now()
		return
	}

	if time.Since(lastOnlineTime) > 15*time.Minute {
		fmt.Println("IdleWatcher: No players for 15 minutes, stopping server.")
		mgr.Stop()
		lastOnlineTime = time.Time{} // Reset after stopping
	}
}

func StartIdleWatcher(mgr *manager.ServerManager) {
	for {
		CheckIdleAndStop(mgr)
		time.Sleep(1 * time.Minute)
	}
}

func GetRemainingTime() string {
	lastOnlineMutex.Lock()
	defer lastOnlineMutex.Unlock()

	if lastOnlineTime.IsZero() {
		return "Server has not been online yet."
	}

	remaining := time.Until(lastOnlineTime.Add(15 * time.Minute))
	if remaining <= 0 {
		return "Server is idle, will stop soon."
	}

	return fmt.Sprintf("Server has been idle for %s, will stop in %s.", time.Since(lastOnlineTime), remaining)
}
