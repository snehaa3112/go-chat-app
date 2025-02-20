package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Client
type Client struct {
	ID      string
	MsgChan chan string
}

// ChatRoom
type ChatRoom struct {
	clients   map[string]*Client
	mu        sync.Mutex
	broadcast chan string
	db        *sql.DB
}

// Initializing database
func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "chat.db")
	if err != nil {
		log.Fatal(err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sender TEXT,
		message TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func NewChatRoom() *ChatRoom {
	chat := &ChatRoom{
		clients:   make(map[string]*Client),
		broadcast: make(chan string),
		db:        initDB(),
	}

	go chat.start()
	return chat
}

func (c *ChatRoom) start() {
	for msg := range c.broadcast {
		c.mu.Lock()
		for id, client := range c.clients {
			select {
			case client.MsgChan <- msg:
			default:
				close(client.MsgChan)
				delete(c.clients, id)
			}
		}
		c.mu.Unlock()
	}
}

func (c *ChatRoom) SaveMessage(sender, message string) {
	_, err := c.db.Exec("INSERT INTO messages (sender, message) VALUES (?, ?)", sender, message)
	if err != nil {
		log.Println("Error saving message:", err)
	}
}

func (c *ChatRoom) Join(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.clients[id]; exists {
		return
	}

	client := &Client{ID: id, MsgChan: make(chan string, 10)}
	c.clients[id] = client
	c.broadcast <- fmt.Sprintf("User %s joined the chat", id)
}

func (c *ChatRoom) Leave(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if client, exists := c.clients[id]; exists {
		close(client.MsgChan)
		delete(c.clients, id)
		c.broadcast <- fmt.Sprintf("User %s left the chat", id)
	}
}

func (c *ChatRoom) SendMessage(id, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.clients[id]; exists {
		fullMessage := fmt.Sprintf("%s: %s", id, message)
		c.broadcast <- fullMessage
		c.SaveMessage(id, message)
	}
}

func (c *ChatRoom) GetMessages(id string, w http.ResponseWriter) {
	c.mu.Lock()
	client, exists := c.clients[id]
	c.mu.Unlock()

	if !exists {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	select {
	case msg, ok := <-client.MsgChan:
		if !ok {
			http.Error(w, "Client disconnected", http.StatusGone)
			return
		}
		fmt.Fprintln(w, msg)
	case <-time.After(10 * time.Second): // Timeout
		http.Error(w, "No new messages", http.StatusNoContent)
	}
}

func (c *ChatRoom) GetChatHistory(w http.ResponseWriter) {
	rows, err := c.db.Query("SELECT sender, message, timestamp FROM messages ORDER BY id DESC LIMIT 10")
	if err != nil {
		http.Error(w, "Could not retrieve messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var sender, message, timestamp string
		rows.Scan(&sender, &message, &timestamp)
		messages = append(messages, fmt.Sprintf("[%s] %s: %s", timestamp, sender, message))
	}

	for _, msg := range messages {
		fmt.Fprintln(w, msg)
	}
}
