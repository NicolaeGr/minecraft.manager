package webui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"electrolit.biz/minecraft.manager/autostop"
	"electrolit.biz/minecraft.manager/manager"
	"github.com/gorilla/websocket"
)

type wsClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

var (
	clientsMu sync.Mutex
	clients   = make(map[*wsClient]struct{})
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (c *wsClient) SafeWriteMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(messageType, data)
}

func broadcastStdout(line string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for c := range clients {
		_ = c.SafeWriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"stdout","line":%q}`, line)))
	}
}

func init() {
	manager.RegisterTerminalListener(broadcastStdout)
}

func StartWebUI(mgr *manager.ServerManager) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlPage)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}
		client := &wsClient{conn: conn}
		defer client.conn.Close()
		clientsMu.Lock()
		clients[client] = struct{}{}
		clientsMu.Unlock()
		defer func() {
			clientsMu.Lock()
			delete(clients, client)
			clientsMu.Unlock()
		}()

		// Send buffer on connect
		for _, line := range manager.GetTerminalBufferTail(50) {
			_ = client.SafeWriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"stdout","line":%q}`, line)))
		}

		// Send periodic server status
		go func() {
			for {
				state, _ := mgr.Status()
				count, max, players, _ := mgr.GetPlayerList()
				// fake this to 0,0,[] for now to avoid RCON dependency
				// count, max, players := 0, 0, []string{}
				countdown := autostop.GetRemainingTime()
				msg := fmt.Sprintf(`{"type":"status","state":"%s","players":%d,"max":%d,"playerNames":%q,"countdown":%q}`,
					state, count, max, players, countdown)
				_ = client.SafeWriteMessage(websocket.TextMessage, []byte(msg))
				time.Sleep(2 * time.Second)
			}
		}()

		// Listen for commands from client
		for {
			_, msg, err := client.conn.ReadMessage()
			if err != nil {
				log.Println("WebSocket read error:", err)
				break
			}
			if len(msg) > 0 && msg[0] == '{' {
				var req struct{ Type, Cmd string }
				_ = json.Unmarshal(msg, &req)
				if req.Type == "rcon" && req.Cmd != "" {
					resp, err := mgr.Command(req.Cmd)
					if err != nil {
						resp = err.Error()
					}
					_ = client.SafeWriteMessage(websocket.TextMessage, fmt.Appendf(nil, `{"type":"rcon","resp":%q}`, resp))
					continue
				}
			}
			cmd := string(msg)
			switch cmd {
			case "start":
				mgr.Start()
			case "stop":
				mgr.Stop()
			case "restart":
				mgr.Restart()
			}
		}
	})

	log.Println("Web UI listening on :8080")
	http.ListenAndServe(":8080", nil)
}

const htmlPage = /** @html */ `<!DOCTYPE html>
<html>
<head>
<title>Minecraft Manager Web UI</title>
<style>
body { font-family: sans-serif; background: #222; color: #eee; }
#terminal { background: #111; color: #0f0; padding: 10px; height: 300px; overflow-y: scroll; font-family: monospace; }
button { margin: 5px; }
#rconModal { display: none; position: fixed; left: 0; top: 0; width: 100vw; height: 100vh; background: rgba(0,0,0,0.7); align-items: center; justify-content: center; }
#rconBox { background: #222; color: #eee; padding: 20px; border-radius: 8px; min-width: 300px; }
</style>
</head>
<body>
<h2>Minecraft Manager Web UI</h2>
<pre id="terminal"></pre>
<div>
<button onclick="send('start')">Start</button>
<button onclick="send('stop')">Stop</button>
<button onclick="send('restart')">Restart</button>
<button onclick="showRcon()">Send RCON</button>
</div>
<div id="status"></div>
<div id="rconModal">
  <div id="rconBox">
    <label>RCON Command:</label><br>
    <input id="rconInput" type="text" style="width:90%" onkeydown="if(event.key==='Enter'){sendRcon()}" autofocus />
    <button onclick="sendRcon()">Send</button>
    <button onclick="hideRcon()">Cancel</button>
    <div id="rconResp" style="margin-top:10px;color:#0af"></div>
  </div>
</div>
<script>
let ws = new WebSocket('ws://' + location.host + '/ws');
ws.onmessage = function(e) {
  let data = JSON.parse(e.data);
  if(data.type === 'status') {
    document.getElementById('status').innerText = 'State: ' + data.state + '\nPlayers: ' + data.players + '/' + data.max + '\n' + data.playerNames + '\nCountdown: ' + data.countdown;
  } else if(data.type === 'stdout') {
    let terminal = document.getElementById('terminal');
    terminal.textContent += data.line + '\n';
    let lines = terminal.textContent.split('\n');
    if (lines.length > 500) {
      terminal.textContent = lines.slice(lines.length - 500).join('\n');
    }
    terminal.scrollTop = terminal.scrollHeight;
  } else if(data.type === 'rcon') {
    document.getElementById('rconResp').innerText = data.resp;
  }
};
function send(cmd) { ws.send(cmd); }
function showRcon() {
  document.getElementById('rconModal').style.display = 'flex';
  document.getElementById('rconInput').value = '';
  document.getElementById('rconResp').innerText = '';
  setTimeout(()=>document.getElementById('rconInput').focus(), 100);
}
function hideRcon() {
  document.getElementById('rconModal').style.display = 'none';
}
function sendRcon() {
  let cmd = document.getElementById('rconInput').value;
  if(cmd) {
    ws.send(JSON.stringify({type:'rcon',cmd:cmd}));
  }
}
</script>
</body>
</html>`
