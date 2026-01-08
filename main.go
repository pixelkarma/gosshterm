package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

var (
	httpPort = flag.String("http", "8000", "HTTP server port")
	sshHost  = flag.String("ssh-host", "localhost", "SSH host")
	sshPort  = flag.String("ssh-port", "22", "SSH port")
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type windowSize struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get credentials from query params, fall back to defaults
	query := r.URL.Query()
	host := query.Get("host")
	port := query.Get("port")
	user := query.Get("user")
	pass := query.Get("pass")

	if host == "" {
		host = *sshHost
	}
	if port == "" {
		port = *sshPort
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// SSH connection config
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to SSH server
	sshAddr := fmt.Sprintf("%s:%s", host, port)
	sshConn, err := ssh.Dial("tcp", sshAddr, config)
	if err != nil {
		log.Printf("SSH dial error: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("SSH connection failed: %v\r\n", err)))
		return
	}
	defer sshConn.Close()

	// Create SSH session
	session, err := sshConn.NewSession()
	if err != nil {
		log.Printf("SSH session error: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("SSH session failed: %v\r\n", err)))
		return
	}
	defer session.Close()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// Request PTY
	if err := session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		log.Printf("PTY request error: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("PTY request failed: %v\r\n", err)))
		return
	}

	// Get stdin/stdout pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		log.Printf("Stdin pipe error: %v", err)
		return
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		log.Printf("Stdout pipe error: %v", err)
		return
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		log.Printf("Stderr pipe error: %v", err)
		return
	}

	// Start shell
	if err := session.Shell(); err != nil {
		log.Printf("Shell start error: %v", err)
		return
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Read from SSH stdout and send to WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			select {
			case <-done:
				return
			default:
				n, err := stdout.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Stdout read error: %v", err)
					}
					return
				}
				if n > 0 {
					if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						log.Printf("WebSocket write error: %v", err)
						return
					}
				}
			}
		}
	}()

	// Read from SSH stderr and send to WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			select {
			case <-done:
				return
			default:
				n, err := stderr.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Stderr read error: %v", err)
					}
					return
				}
				if n > 0 {
					if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						log.Printf("WebSocket write error: %v", err)
						return
					}
				}
			}
		}
	}()

	// Read from WebSocket and send to SSH stdin
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			close(done)
			break
		}

		// Check if it's a resize message (JSON)
		if msgType == websocket.TextMessage {
			var resize windowSize
			if err := json.Unmarshal(msg, &resize); err == nil && resize.Rows > 0 && resize.Cols > 0 {
				session.WindowChange(resize.Rows, resize.Cols)
				continue
			}
		}

		// Otherwise, send as input
		if _, err := stdin.Write(msg); err != nil {
			log.Printf("SSH stdin write error: %v", err)
			close(done)
			break
		}
	}

	session.Close()
	wg.Wait()
}

func main() {
	flag.Parse()

	log.Printf("SSH target: %s:%s", *sshHost, *sshPort)

	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Printf("Server starting on :%s", *httpPort)
	log.Fatal(http.ListenAndServe(":"+*httpPort, nil))
}
