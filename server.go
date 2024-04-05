package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

type Lobby struct {
	clients    map[*Client]bool
	broadcast  chan string
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	conn    net.Conn
	name    string
	lobby   *Lobby
	message chan string // Channel for sending messages to the client
}

// Сreating new lobby
func NewLobby() *Lobby {
	return &Lobby{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan string),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Running the lobby
func (l *Lobby) Run() {
	for {
		select {
		case client := <-l.register:
			l.clients[client] = true
			log.Printf("Client %s joined\n", client.name)
			go client.Read()
			go client.Write()
		case client := <-l.unregister:
			l.unregisterClient(client)
		case message := <-l.broadcast:
			for client := range l.clients {
				client.message <- message
			}
		}
	}
}

// Handling unregistered user
func (l *Lobby) unregisterClient(client *Client) {
	if _, ok := l.clients[client]; ok {
		delete(l.clients, client)
		log.Printf("Client %s left\n", client.name)
		close(client.message)
		client.conn.Close()
	}
}

// Creating new client
func NewClient(conn net.Conn, lobby *Lobby) *Client {
	// Read the username from the client
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	name := scanner.Text()

	return &Client{
		conn:    conn,
		name:    name,
		lobby:   lobby,
		message: make(chan string), // Initialize the message channel
	}
}

// Reading messages from the client
func (c *Client) Read() {
	defer func() {
		c.lobby.unregister <- c
	}()
	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		message := scanner.Text()
		c.lobby.broadcast <- fmt.Sprintf("[%s] %s: %s\n", time.Now().Format("2006-01-02 15:04:05"), c.name, message)
	}
}

// Writing messages to the client
func (c *Client) Write() {
	defer func() {
		c.lobby.unregister <- c
	}()
	for message := range c.message {
		_, err := fmt.Fprintf(c.conn, "%s\n", message)
		if err != nil {
			log.Printf("Error sending message to %s: %v", c.name, err)
			return
		}
	}
}

func main() {
	l := NewLobby()
	go l.Run()

	listener, err := net.Listen("tcp", ":3333")
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer listener.Close()

	log.Println("Server started, listening on :3333")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		client := NewClient(conn, l)
		l.register <- client
	}
}
