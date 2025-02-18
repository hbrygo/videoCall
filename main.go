package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	conn *websocket.Conn
	id   string
}

type Message struct {
	Type string `json:"type"`
	Data string `json:"data"`
	From string `json:"from"`
	To   string `json:"to"`
	Name string `json:"name"`
}

var (
	clients = make(map[string]*Client)
	mutex   sync.RWMutex
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	clientID := r.URL.Query().Get("id")
	clientName := r.URL.Query().Get("name")
	fmt.Printf("Client ID: %s, Name: %s\n", clientID, clientName)
	if clientID == "" {
		clientID = fmt.Sprintf("user-%d", len(clients))
	}
	log.Printf("Client connected with ID: %s", clientID)

	client := &Client{
		conn: conn,
		id:   clientID,
	}

	mutex.Lock()
	clients[clientID] = client
	mutex.Unlock()

	// Broadcast new user immediately after connection
	broadcastNewUser(clientID, clientName)
	log.Printf("Total connected clients: %d", len(clients))

	defer func() {
		mutex.Lock()
		delete(clients, clientID)
		mutex.Unlock()
		// Broadcast user disconnection
		broadcastMessage(Message{
			Type: "user_disconnected",
			Data: clientID,
		})
		conn.Close()
	}()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Client %s disconnected: %v", clientID, err)
			break
		}

		msg.From = clientID
		log.Printf("Message from %s to %s: %s", msg.From, msg.To, msg.Type)

		if msg.To != "" {
			// Direct message
			if targetClient, ok := clients[msg.To]; ok {
				if err := targetClient.conn.WriteJSON(msg); err != nil {
					log.Printf("Error sending to client %s: %v", msg.To, err)
				}
			}
		}
	}
}

func broadcastMessage(msg Message) {
	mutex.RLock()
	defer mutex.RUnlock()

	for _, client := range clients {
		if err := client.conn.WriteJSON(msg); err != nil {
			log.Printf("Broadcast error to %s: %v", client.id, err)
		}
	}
}

func broadcastNewUser(newClientID string, newClientName string) {
	broadcastMessage(Message{
		Type: "new_user",
		Data: newClientID,
		Name: newClientName,
	})
}

func sendAllUsers(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	defer mutex.RUnlock()

	var users []string
	for id := range clients {
		users = append(users, id)
	}

	w.Header().Set("Content-Type", "application/json")
	// send user id and user name in response
	json.NewEncoder(w).Encode(map[string]interface{}{"users": clients})
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/users", sendAllUsers)
	http.Handle("/", http.FileServer(http.Dir("static")))

	fmt.Println("Server starting at https://localhost:8080")
	log.Fatal(http.ListenAndServeTLS(":8080", "cert/cert.csr", "cert/cert.key", nil))
}
