package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Msg struct {
	Username string `json:"username"`
	Content  string `json:"content"`
}

var db *sql.DB

// --------------------
// DB
// --------------------
func initDB() {
	var err error
	db, err = sql.Open("sqlite", "chat.db")
	if err != nil {
		log.Fatal(err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT,
		content TEXT,
		created_at DATETIME
	);
	`

	if _, err := db.Exec(query); err != nil {
		log.Fatal(err)
	}
}

func saveMessage(username, content string) {
	_, err := db.Exec(
		"INSERT INTO messages(username, content, created_at) VALUES (?, ?, ?)",
		username,
		content,
		time.Now(),
	)
	if err != nil {
		log.Println("DB error:", err)
	}
}

// --------------------
// WebSocket (ONLY save message)
// --------------------
func wsHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	log.Println("Connected:", username)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Disconnected:", username)
			return
		}

		// فقط ذخیره
		saveMessage(username, string(msg))
	}
}

// --------------------
// History (ONLY own messages)
// --------------------
func historyHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(
		"SELECT content FROM messages WHERE username = ? ORDER BY id ASC",
		username,
	)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	messages := []Msg{}

	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err == nil {
			messages = append(messages, Msg{
				Username: username,
				Content:  content,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// --------------------
func main() {
	initDB()

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/history", historyHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Println("🚀 Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
