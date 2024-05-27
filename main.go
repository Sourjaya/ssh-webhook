package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/joho/godotenv"
	"github.com/teris-io/shortid"
	gossh "golang.org/x/crypto/ssh"
)

func startSSHServer() error {
	// respCh := make(chan string)
	// go func() {
	// 	time.Sleep(time.Second * 3)
	// 	id, _ := shortid.Generate()
	// 	respCh <- "http://webhooker.com/" + id

	// 	time.Sleep(time.Second * 10)
	// 	for {
	// 		time.Sleep(time.Second * 2)
	// 		respCh <- "received data from hook"
	// 	}
	// }()
	handler := NewSSHHandler()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load environment variables")
	}

	sshPort := os.Getenv("SSH_PORT")

	if sshPort == "" {
		log.Fatal("SSH PORT not found in the environment")
	}

	server := ssh.Server{
		Addr:    ":" + sshPort,
		Handler: handler.handleSSHSession,
		ServerConfigCallback: func(ctx ssh.Context) *gossh.ServerConfig {
			cfg := &gossh.ServerConfig{
				ServerVersion: "SSH-2.0-sendit",
			}
			cfg.Ciphers = []string{"chacha20-poly1305@openssh.com"}
			return cfg
		},
		PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
			return true
		},
	}
	b, err := os.ReadFile("keys/privatekey")
	if err != nil {
		log.Fatal(err)
	}
	privateKey, err := gossh.ParsePrivateKey(b)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}
	server.AddHostKey(privateKey)
	fmt.Println("Server running at port: ", sshPort)
	return server.ListenAndServe()
}

var clients sync.Map

type HTTPHandler struct {
}

func (h *HTTPHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ch, ok := clients.Load(id)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("client id not found"))
		return
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Body.Close()

	ch.(chan string) <- string(b)
}

func startHTTPServer() error {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load environment variables")
	}

	httpPort := os.Getenv("HTTP_PORT")

	if httpPort == "" {
		log.Fatal("HTTP PORT not found in the environment")
	}
	router := http.NewServeMux()

	handler := &HTTPHandler{}
	router.HandleFunc("/{id}/*", handler.handleWebhook)
	fmt.Println("HTTP Server running at port: ", httpPort)
	return http.ListenAndServe(":"+httpPort, router)
}

func main() {
	go startSSHServer()
	startHTTPServer()
}

type SSHHandler struct {
	channels map[string]chan string
}

func NewSSHHandler() *SSHHandler {
	return &SSHHandler{
		channels: make(map[string]chan string),
	}
}

func (h *SSHHandler) handleSSHSession(session ssh.Session) {
	// forwardURL := session.RawCommand()
	// _ = forwardURL
	// webhookURL := <-h.respCh
	cmd := session.RawCommand()
	if cmd == "init" {
		id := shortid.MustGenerate()
		webhookURL := "http://localhost:5000/" + id + "\n"
		resp := fmt.Sprintf("webhook url %sssh localhost -p 2222 %s | curl -X POST -H 'Content-Type: application/json' -d @- http://localhost:3000/payment/webhook\n", webhookURL, id)
		session.Write([]byte(resp))
		respCh := make(chan string, 1)
		h.channels[id] = respCh
		// respCh := make(chan string)
		clients.Store(id, respCh)
	}
	if len(cmd) > 0 && cmd != "init" {
		respCh, ok := h.channels[cmd]
		if !ok {
			session.Write([]byte("invalid webhook id\n"))
			return
		}
		for data := range respCh {
			session.Write([]byte(data + "\n"))
		}
	}
	// for {
	// 	n, err := session.Read(buf)
	// 	if err == io.EOF {
	// 		break
	// 	}
	// 	log.Fatal(err)
	// 	fmt.Println(string(buf[:n]))
	// }
}
