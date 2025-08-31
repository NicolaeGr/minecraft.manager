package manager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	rconlib "github.com/jltobler/go-rcon"
)

type ServerManager struct {
	opts Options

	mu       sync.RWMutex
	state    State
	lastErr  error
	cmd      *exec.Cmd
	superTok func()

	rcon *rconlib.Conn
}

func NewServerManager(workDir string) *ServerManager {
	opts := Options{WorkingDir: workDir}

	opts.RunScript = "./run.sh"
	if opts.DialTimeout == 0 {
		opts.DialTimeout = 3 * time.Second
	}
	if opts.CommandTimeout == 0 {
		opts.CommandTimeout = 4 * time.Second
	}
	if opts.MinBackoff == 0 {
		opts.MinBackoff = 500 * time.Millisecond
	}
	if opts.MaxBackoff == 0 {
		opts.MaxBackoff = 10 * time.Second
	}

	props, err := parseServerProperties(filepath.Join(workDir, "server.properties"))
	if err == nil {
		if port, ok := props["rcon.port"]; ok {
			opts.RCONPort, _ = strconv.Atoi(port)
			fmt.Println("RCON port loaded from server.properties, it's value is:", opts.RCONPort)
		}
		if pass, ok := props["rcon.password"]; ok {
			opts.RCONPassword = pass
			fmt.Println("RCON password loaded from server.properties, it's value is:", pass)
		}
		if host, ok := props["server-ip"]; ok && host != "" {
			opts.RCONHost = host
			fmt.Println("RCON host loaded from server.properties, it's value is:", host)
		}
	}
	if opts.RCONHost == "" {
		opts.RCONHost = "127.0.0.1"
	}
	if opts.RCONPort == 0 {
		opts.RCONPort = 25575
	}

	return &ServerManager{
		opts:  opts,
		state: StateStopped,
		rcon:  nil, // rcon is now initialized after server is running
	}
}

func (sm *ServerManager) Start() error {
	sm.mu.Lock()
	if sm.state != StateStopped && sm.state != StateCrashed {
		sm.mu.Unlock()
		return fmt.Errorf("cannot start in state %s", sm.state)
	}
	sm.state = StateStarting
	sm.lastErr = nil
	sm.mu.Unlock()

	if sm.opts.WorkingDir != "" {
		if err := os.Chdir(sm.opts.WorkingDir); err != nil {
			sm.setCrashed(err)
			return fmt.Errorf("chdir: %w", err)
		}
	}
	if _, err := os.Stat(sm.opts.RunScript); err != nil {
		sm.setCrashed(err)
		return fmt.Errorf("run script not found: %w", err)
	}

	cmd := exec.Command(sm.opts.RunScript)
	stdoutWriter := NewTerminalWriter(sm) // Only write to terminal buffer, not app stdout
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stdoutWriter

	if err := cmd.Start(); err != nil {
		sm.setCrashed(err)
		return fmt.Errorf("start server: %w", err)
	}

	cancel := func() {
		_ = cmd.Process.Kill()
	}
	sm.mu.Lock()
	sm.cmd = cmd
	sm.superTok = cancel
	sm.mu.Unlock()

	go sm.watchProcess(cmd)

	return nil
}

func (sm *ServerManager) Stop() error {
	sm.mu.Lock()
	if sm.state == StateStopped {
		sm.mu.Unlock()
		return nil
	}
	sm.state = StateStopping
	cancel := sm.superTok
	sm.mu.Unlock()

	if sm.rcon != nil {
		_, err := sm.rcon.SendCommand("stop")
		if err != nil {
			fmt.Println("Error sending stop command via RCON:", err)
		}
	}

	var waitErr error
	sm.mu.RLock()
	cmd := sm.cmd
	sm.mu.RUnlock()
	if cmd != nil && cmd.Process != nil {
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case waitErr = <-done:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
			waitErr = fmt.Errorf("timeout waiting for process to exit")
		}
	}

	if cancel != nil {
		cancel()
	}

	sm.mu.Lock()
	sm.state = StateStopped
	sm.mu.Unlock()
	return waitErr
}

func (sm *ServerManager) Restart() error {
	_ = sm.Stop()
	return sm.Start()
}

func (sm *ServerManager) Status() (State, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state, sm.lastErr
}

func (sm *ServerManager) setCrashed(err error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state = StateCrashed
	sm.lastErr = err
}

func (sm *ServerManager) setRunning() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.state == StateStarting || sm.state == StateStopping {
		sm.state = StateRunning
		if sm.rcon == nil && sm.state == StateRunning {
			time.Sleep(2 * time.Second)
			addr := fmt.Sprintf("rcon://%s:%d", sm.opts.RCONHost, sm.opts.RCONPort)
			conn, err := rconlib.Dial(addr, sm.opts.RCONPassword)
			if err != nil {
				fmt.Println("Failed to dial RCON:", err)
			} else {
				sm.rcon = conn
			}
		}
	}
}

func (sm *ServerManager) watchProcess(cmd *exec.Cmd) {
	err := cmd.Wait()
	if err != nil {
		sm.setCrashed(fmt.Errorf("server exited with error: %w", err))
	} else {
		sm.mu.Lock()
		sm.state = StateStopped
		sm.mu.Unlock()
	}
}

func (sm *ServerManager) Command(cmd string) (string, error) {
	fmt.Println("Executing RCON command:", cmd)
	if sm.rcon == nil || sm.rcon.IsClosed() {
		addr := fmt.Sprintf("%s:%d", sm.opts.RCONHost, sm.opts.RCONPort)
		conn, err := rconlib.Dial(addr, sm.opts.RCONPassword)
		if err != nil {
			return "", fmt.Errorf("RCON not connected: %w", err)
		}
		sm.rcon = conn
	}
	resp, err := sm.rcon.SendCommand(cmd)
	if err != nil {
		return "", err
	}
	return resp, nil
}

func (sm *ServerManager) GetPlayerList() (count, max int, players []string, err error) {
	if sm.rcon == nil {
		return 0, 0, nil, fmt.Errorf("RCON not connected")
	}
	out, err := sm.Command("list")
	if err != nil {
		fmt.Println("RCON list command failed:", err)
		return 0, 0, nil, err
	}
	// Try to parse output for player list
	var n, mx int
	var names []string
	if strings.Contains(out, "players online:") {
		// Example: There are 1/20 players online: Player1
		parts := strings.SplitN(out, ":", 2)
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "There are %d/%d players online", &n, &mx)
			namesStr := strings.TrimSpace(parts[1])
			if namesStr != "" {
				names = strings.Split(namesStr, ", ")
			}
		}
	}
	return n, mx, names, nil
}

type LineListener func(line string)

type RingBuffer struct {
	lines []string
	size  int
	pos   int
	full  bool
	sync.RWMutex
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{lines: make([]string, size), size: size}
}

func (b *RingBuffer) Add(line string) {
	b.Lock()
	defer b.Unlock()
	b.lines[b.pos] = line
	b.pos = (b.pos + 1) % b.size
	if b.pos == 0 {
		b.full = true
	}
}

func (b *RingBuffer) GetAll() []string {
	b.RLock()
	defer b.RUnlock()
	var out []string
	if b.full {
		out = append(out, b.lines[b.pos:]...)
		out = append(out, b.lines[:b.pos]...)
	} else {
		out = append(out, b.lines[:b.pos]...)
	}
	return out
}

type terminalHub struct {
	buffer    *RingBuffer
	listeners []LineListener
	sync.RWMutex
}

func newTerminalHub(size int) *terminalHub {
	return &terminalHub{buffer: NewRingBuffer(size)}
}

func (h *terminalHub) AddListener(l LineListener) {
	h.Lock()
	h.listeners = append(h.listeners, l)
	h.Unlock()
}

func (h *terminalHub) WriteLine(line string) {
	h.buffer.Add(line)
	h.RLock()
	for _, l := range h.listeners {
		l(line)
	}
	h.RUnlock()
}

func (h *terminalHub) GetBuffer() []string {
	return h.buffer.GetAll()
}

type TerminalWriter struct {
	mgr *ServerManager
}

func NewTerminalWriter(mgr *ServerManager) *TerminalWriter {
	return &TerminalWriter{mgr: mgr}
}

func (w *TerminalWriter) Write(p []byte) (int, error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line != "" {
			defaultTerminalHub.WriteLine(line)
			if w.mgr == nil || w.mgr.state != StateStarting {
				continue
			}
			if strings.Contains(line, "Done ") && strings.Contains(line, "For help, type \"help\"") {
				w.mgr.setRunning()
			}
		}
	}
	return len(p), nil
}

var defaultTerminalHub = newTerminalHub(1000)

func WriteToTerminal(line string) {
	defaultTerminalHub.WriteLine(line)
}

func RegisterTerminalListener(l LineListener) {
	defaultTerminalHub.AddListener(l)
}

func GetTerminalBuffer() []string {
	return defaultTerminalHub.GetBuffer()
}
