package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gliderlabs/ssh"
	"github.com/joho/godotenv"
	gossh "golang.org/x/crypto/ssh"
)

func main() {
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
		Handler: handleSSHSession,
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
	log.Fatal(server.ListenAndServe())
}

func handleSSHSession(session ssh.Session) {
	session.R
	fmt.Println("Hello")
}
