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

func CheckIdleAndStop(mgr *manager.ServerManager) {
	lastOnlineMutex.Lock()
	defer lastOnlineMutex.Unlock()

	status := mgr.Status()
	if status != "running" {
		return
	}

	count, _, _, err := mgr.GetPlayerList()
	if err != nil {
		fmt.Println("IdleWatcher: error getting player list:", err)
		return
	}

	if count > 0 {
		lastOnlineTime = time.Now()
		return
	}

	if lastOnlineTime.IsZero() {
		lastOnlineTime = time.Now()
		return
	}

	if time.Since(lastOnlineTime) > 15*time.Minute {
		fmt.Println("IdleWatcher: No players for 15 minutes, stopping server.")
		mgr.Stop()
	}
}

func StartIdleWatcher(mgr *manager.ServerManager) {
	for {
		CheckIdleAndStop(mgr)
		time.Sleep(1 * time.Minute)
	}
}
