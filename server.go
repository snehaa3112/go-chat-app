package main

import (
	"fmt"
	"net/http"
	"os"
)

// CORS middleware
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

var chatRoom = NewChatRoom()

func joinChatHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	chatRoom.Join(id)
	fmt.Fprintf(w, "User %s joined the chat\n", id)
}

func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	message := r.URL.Query().Get("message")

	if id == "" || message == "" {
		http.Error(w, "ID and message required", http.StatusBadRequest)
		return
	}

	chatRoom.SendMessage(id, message)
	fmt.Fprintf(w, "Message sent by %s\n", id)
}

func leaveChatHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	chatRoom.Leave(id)
	fmt.Fprintf(w, "User %s left the chat\n", id)
}

func getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	chatRoom.GetMessages(id, w)
}

func getChatHistoryHandler(w http.ResponseWriter, r *http.Request) {
	chatRoom.GetChatHistory(w)
}

func main() {
	port := os.Getenv("PORT") 
	if port == "" {
		port = "8080" // Default for local testing
	}

	http.HandleFunc("/join", enableCORS(joinChatHandler))
	http.HandleFunc("/send", enableCORS(sendMessageHandler))
	http.HandleFunc("/leave", enableCORS(leaveChatHandler))
	http.HandleFunc("/messages", enableCORS(getMessagesHandler))
	http.HandleFunc("/history", enableCORS(getChatHistoryHandler))

	fmt.Println("Server started at :" + port)
	http.ListenAndServe(":"+port, nil)
}
