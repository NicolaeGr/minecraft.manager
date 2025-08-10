package manager

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ConsoleHandler func(line string) (result interface{}, handled bool)

type ServerManager struct {
	cmd              *exec.Cmd
	stdin            io.WriteCloser
	stdout           io.ReadCloser
	mu               sync.Mutex
	isActive         bool
	isReady          bool
	ShowServerStdout bool
	workingPath      string

	handlers       map[string]ConsoleHandler
	handlerResults map[string]interface{}
}

func NewServerManager() *ServerManager {
	return &ServerManager{
		handlers:       make(map[string]ConsoleHandler),
		handlerResults: make(map[string]interface{}),
	}
}

func (sm *ServerManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.isActive {
		return fmt.Errorf("server already running")
	}
	if sm.workingPath != "" {
		if err := os.Chdir(sm.workingPath); err != nil {
			return fmt.Errorf("failed to change working directory: %w", err)
		}
	}
	cmd := exec.Command("./run.sh", "nogui")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	sm.cmd = cmd
	sm.stdin = stdin
	sm.stdout = stdout
	sm.isActive = true
	sm.isReady = false
	go sm.readConsole()
	return nil
}

func (sm *ServerManager) RegisterHandler(key string, handler ConsoleHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.handlers[key] = handler
}

func (sm *ServerManager) RemoveHandler(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.handlers, key)
	delete(sm.handlerResults, key)
}

func (sm *ServerManager) GetHandlerResult(key string) interface{} {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.handlerResults[key]
}

func (sm *ServerManager) readConsole() {
	scanner := bufio.NewScanner(sm.stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if sm.ShowServerStdout {
			fmt.Println(line)
		}
		if strings.Contains(line, "Done") && strings.Contains(line, "For help, type \"help\"") {
			sm.mu.Lock()
			sm.isReady = true
			sm.mu.Unlock()
		}

		// Call all registered handlers
		sm.mu.Lock()
		for key, handler := range sm.handlers {
			if result, handled := handler(line); handled {
				sm.handlerResults[key] = result
			}
		}
		sm.mu.Unlock()
	}
}

func (sm *ServerManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if !sm.isActive {
		return fmt.Errorf("server not running")
	}
	if sm.cmd != nil && sm.cmd.Process != nil {
		_, err := sm.stdin.Write([]byte("stop\n"))
		if err != nil {
			return err
		}
		err = sm.cmd.Wait()
		sm.isActive = false
		sm.isReady = false

		return err
	}
	sm.isActive = false
	sm.isReady = false

	return nil
}

func (sm *ServerManager) Status() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.isActive {
		if sm.isReady {
			return "running"
		}
		return "starting"
	}
	return "stopped"
}

func (sm *ServerManager) Restart() error {
	err := sm.Stop()
	if err != nil {
		return err
	}
	sm.isActive = false
	sm.isReady = false

	return sm.Start()
}

func (sm *ServerManager) SendCommand(cmd string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if !sm.isActive {
		return fmt.Errorf("server not running")
	}
	_, err := sm.stdin.Write([]byte(cmd + "\n"))
	return err
}

func (sm *ServerManager) GetPlayerList() (count int, max int, players []string, err error) {
	sm.mu.Lock()
	if !sm.isActive || !sm.isReady {
		sm.mu.Unlock()
		return 0, 0, nil, fmt.Errorf("server not running or not ready")
	}
	key := "playerlist_temp"
	responseChan := make(chan struct {
		count   int
		max     int
		players []string
		err     error
	}, 1)

	handler := func(line string) (interface{}, bool) {
		if strings.Contains(line, "players online:") {
			var n, m int
			var names []string

			// Try to match both known formats
			re1 := regexp.MustCompile(`There are (\d+) of a max of (\d+) players online:`)
			re2 := regexp.MustCompile(`There are (\d+)/(\d+) players online:`)

			if matches := re1.FindStringSubmatch(line); len(matches) == 3 {
				n = parseInt(matches[1])
				m = parseInt(matches[2])
			} else if matches := re2.FindStringSubmatch(line); len(matches) == 3 {
				n = parseInt(matches[1])
				m = parseInt(matches[2])
			} else {
			}
			lastIdx := strings.LastIndex(line, ":")
			if lastIdx != -1 && lastIdx+1 < len(line) {
				nameStr := strings.TrimSpace(line[lastIdx+1:])
				if nameStr != "" {
					names = strings.Split(nameStr, ", ")
				}
			}
			responseChan <- struct {
				count   int
				max     int
				players []string
				err     error
			}{n, m, names, nil}
			return nil, true
		}
		return nil, false
	}
	sm.handlers[key] = handler
	sm.mu.Unlock()

	err = sm.SendCommand("list")
	if err != nil {
		sm.RemoveHandler(key)
		return 0, 0, nil, err
	}

	select {
	case resp := <-responseChan:
		sm.RemoveHandler(key)
		return resp.count, resp.max, resp.players, resp.err
	case <-time.After(4 * time.Second):
		sm.RemoveHandler(key)
		return 0, 0, nil, fmt.Errorf("timeout waiting for player list")
	}
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		fmt.Println("parseInt error:", err, "input:", s)
		return 0
	}
	return n
}
