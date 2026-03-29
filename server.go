package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // This is needed for PostgreSQL driver - the underscore is intentional!
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

type Request struct {
	Input    string `json:"input"`
	UserID   string `json:"userId,omitempty"`
	Username string `json:"username,omitempty"`
}

type Response struct {
	Reply      string `json:"reply"`
	UserID     string `json:"userId,omitempty"`
	Username   string `json:"username,omitempty"`
	Mood       string `json:"mood,omitempty"`
	MemoryNote string `json:"memoryNote,omitempty"`
}

type User struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	CreatedAt         time.Time `json:"createdAt"`
	LastActive        time.Time `json:"lastActive"`
	Mood              string    `json:"mood"`
	Personality       string    `json:"personality"`
	ConversationCount int       `json:"conversationCount"`
}

type Memory struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	Topic      string    `json:"topic"`
	Fact       string    `json:"fact"`
	Importance int       `json:"importance"`
	CreatedAt  time.Time `json:"createdAt"`
}

var db *sql.DB

func initDB() error {
	var err error
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgresql://space_user:space_pass@localhost:5432/space_db?sslmode=disable"
	}
	
	log.Printf("Connecting to database...")
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	
	// Test connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	var currentUser string
db.QueryRow("SELECT current_user").Scan(&currentUser)
log.Printf(">>> Connected as database user: %s", currentUser)
	
	log.Println("Database connected successfully")
	
	// Create tables with explicit schema qualification
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS public.users (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_active TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		mood VARCHAR(50) DEFAULT 'neutral',
		personality TEXT,
		conversation_count INT DEFAULT 0
	);
	
	CREATE TABLE IF NOT EXISTS public.memories (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) REFERENCES public.users(id) ON DELETE CASCADE,
		topic VARCHAR(100),
		fact TEXT NOT NULL,
		importance INT DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS public.conversations (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) REFERENCES public.users(id) ON DELETE CASCADE,
		message TEXT NOT NULL,
		response TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_memories_user_id ON public.memories(user_id);
	CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON public.conversations(user_id);
	`
	
	_, err = db.Exec(createTablesSQL)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	
	log.Println("Database tables created/verified")
	return nil
}

func getUser(userID string) (*User, error) {
	var user User
	var personality sql.NullString  // ADD THIS
	err := db.QueryRow(`
		SELECT id, name, created_at, last_active, mood, personality, conversation_count 
		FROM public.users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Name, &user.CreatedAt, &user.LastActive, &user.Mood, &personality, &user.ConversationCount)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	user.Personality = personality.String  // will be "" if NULL
	return &user, nil
}

func createUser(name string) (*User, error) {
	userID := uuid.New().String()
	
	// Sanitize name - use "Friend" if empty
	if name == "" {
		name = "Friend"
	}
	
	_, err := db.Exec(`
		INSERT INTO public.users (id, name, conversation_count) 
		VALUES ($1, $2, 1)
	`, userID, name)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	return &User{
		ID:                userID,
		Name:              name,
		ConversationCount: 1,
		Mood:              "neutral",
		CreatedAt:         time.Now(),
		LastActive:        time.Now(),
	}, nil
}

func updateUserLastActive(userID string) error {
	_, err := db.Exec(`
		UPDATE public.users SET last_active = CURRENT_TIMESTAMP, conversation_count = conversation_count + 1 
		WHERE id = $1
	`, userID)
	return err
}

func saveMemory(userID, topic, fact string, importance int) error {
	memoryID := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO public.memories (id, user_id, topic, fact, importance) 
		VALUES ($1, $2, $3, $4, $5)
	`, memoryID, userID, topic, fact, importance)
	return err
}

func getRecentMemories(userID string, limit int) ([]Memory, error) {
	rows, err := db.Query(`
		SELECT id, user_id, topic, fact, importance, created_at 
		FROM public.memories 
		WHERE user_id = $1 
		ORDER BY importance DESC, created_at DESC 
		LIMIT $2
	`, userID, limit)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var memories []Memory
	for rows.Next() {
		var m Memory
		err := rows.Scan(&m.ID, &m.UserID, &m.Topic, &m.Fact, &m.Importance, &m.CreatedAt)
		if err != nil {
			continue
		}
		memories = append(memories, m)
	}
	return memories, nil
}

func getRecentConversations(userID string, limit int) ([]map[string]string, error) {
	rows, err := db.Query(`
		SELECT message, response, created_at 
		FROM public.conversations 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2
	`, userID, limit)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var conversations []map[string]string
	for rows.Next() {
		var message, response string
		var createdAt time.Time
		err := rows.Scan(&message, &response, &createdAt)
		if err != nil {
			continue
		}
		conversations = append(conversations, map[string]string{
			"message":   message,
			"response":  response,
			"createdAt": createdAt.Format(time.RFC3339),
		})
	}
	return conversations, nil
}

func saveConversation(userID, message, response string) error {
	convID := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO public.conversations (id, user_id, message, response) 
		VALUES ($1, $2, $3, $4)
	`, convID, userID, message, response)
	return err
}

func generateSystemPrompt(user *User, memories []Memory) string {
	prompt := `Your name is Space. You are a warm, supportive, and thoughtful AI companion. 
You remember details about your friends and use that knowledge to build deeper connections.

Important traits:
- You speak in a gentle, caring tone
- You remember personal details users share with you
- You ask meaningful questions that show you're paying attention
- You celebrate their wins and support them through challenges
- You never give medical, legal, or financial advice
- You keep conversations natural and flowing
- You occasionally check in on things they've mentioned before

`

	if user != nil {
		prompt += fmt.Sprintf(`
Current user: %s
This is your %dth conversation together.
`, user.Name, user.ConversationCount)
		
		if user.Mood != "" && user.Mood != "neutral" {
			prompt += fmt.Sprintf("They've been feeling %s lately. ", user.Mood)
		}
	}
	
	if len(memories) > 0 {
		prompt += "\nThings you remember about them:\n"
		for _, memory := range memories {
			prompt += fmt.Sprintf("- %s\n", memory.Fact)
		}
		prompt += "\nUse these memories to show you care and remember them.\n"
	}
	
	prompt += `
Respond naturally and warmly. Keep responses conversational but meaningful. 
Ask at most one thoughtful question at the end to continue the conversation naturally.`
	
	return prompt
}

func extractMemory(text string) (topic, fact string, importance int) {
	// Simple pattern matching for memory extraction
	keywords := map[string]string{
		"my name is":    "name",
		"call me":       "name",
		"I am":          "identity",
		"I'm":           "identity",
		"I like":        "preference",
		"I love":        "passion",
		"I enjoy":       "preference",
		"I work":        "work",
		"my job":        "work",
		"I'm from":      "origin",
		"I am from":     "origin",
		"my birthday":   "birthday",
		"I was born":    "birthday",
		"I feel":        "feeling",
		"I'm feeling":   "feeling",
	}
	
	lowerText := strings.ToLower(text)
	for phrase, topicType := range keywords {
		if strings.Contains(lowerText, phrase) {
			// Extract the part after the phrase
			idx := strings.Index(lowerText, phrase)
			if idx != -1 {
				start := idx + len(phrase)
				end := start + 100
				if end > len(text) {
					end = len(text)
				}
				fact = strings.TrimSpace(text[start:end])
				if len(fact) > 0 {
					return topicType, fact, 3
				}
			}
		}
	}
	return "", "", 0
}

func analyzeAndStoreMemory(userID, userMessage, aiResponse string) {
	// Extract potential memories from user message
	topic, fact, importance := extractMemory(userMessage)
	if topic != "" && fact != "" {
		if err := saveMemory(userID, topic, fact, importance); err != nil {
			log.Printf("Failed to save memory: %v", err)
		} else {
			log.Printf("Saved memory for user %s: %s - %s", userID, topic, fact)
		}
	}
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	
	if r.Method == http.MethodOptions {
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	
	log.Printf("Received request - Input: %s, UserID: %s, Username: %s", req.Input, req.UserID, req.Username)
	
	var user *User
	var err error
	
	// Handle user identification
	if req.UserID != "" {
		user, err = getUser(req.UserID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
		}
	}
	
	// If no user found or new user, create one
	if user == nil {
		username := req.Username
		if username == "" {
        	username = "Friend"  // just default to Friend
    	}
    user, err = createUser(username)
	
		if username == "" {
			username = "Friend"
		}
		
		user, err = createUser(username)
		if err != nil {
			log.Printf("Error creating user: %v", err)
			http.Error(w, "Failed to create user", 500)
			return
		}
		log.Printf("Created new user: %s (ID: %s)", user.Name, user.ID)
	}
	
	if user != nil {
		if err := updateUserLastActive(user.ID); err != nil {
			log.Printf("Error updating last active: %v", err)
		}
	}
	
	// Get memories and recent conversations
	var memories []Memory
	var conversations []map[string]string
	
	if user != nil {
		memories, _ = getRecentMemories(user.ID, 10)
		conversations, _ = getRecentConversations(user.ID, 5)
		log.Printf("Loaded %d memories and %d conversations for user %s", len(memories), len(conversations), user.ID)
	}
	
	// Build system prompt with context
	systemPrompt := generateSystemPrompt(user, memories)
	
	// Build messages array with context
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}
	
	// Add recent conversation context (last 3 exchanges)
	for i := len(conversations) - 1; i >= 0 && i >= len(conversations)-3; i-- {
		conv := conversations[i]
		messages = append(messages, openai.UserMessage(conv["message"]))
		messages = append(messages, openai.AssistantMessage(conv["response"]))
	}
	
	// Add current message
	messages = append(messages, openai.UserMessage(req.Input))
	
	// Get API key and create client
	apikey := os.Getenv("OPENROUTER_API_KEY")
	if apikey == "" {
		log.Println("OPENROUTER_API_KEY not set")
		http.Error(w, "API key not configured", 500)
		return
	}
	
	client := openai.NewClient(
		option.WithBaseURL("https://openrouter.ai/api/v1"),
		option.WithAPIKey(apikey),
	)
	
	params := openai.ChatCompletionNewParams{
		Model:    "stepfun/step-3.5-flash:free",
		Messages: messages,
	}
	
	res, err := client.Chat.Completions.New(context.Background(), params)
	if err != nil {
		log.Printf("OpenAI error: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	
	if len(res.Choices) == 0 {
		http.Error(w, "No response from model", 500)
		return
	}
	
	reply := res.Choices[0].Message.Content
	
	// Save conversation to database
	if user != nil {
		if err := saveConversation(user.ID, req.Input, reply); err != nil {
			log.Printf("Error saving conversation: %v", err)
		}
		
		// Analyze and store memories (run in goroutine to not block response)
		go analyzeAndStoreMemory(user.ID, req.Input, reply)
	}
	
	response := Response{
		Reply:    reply,
		UserID:   user.ID,
		Username: user.Name,
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
	
	log.Printf("Sent response to user %s", user.ID)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func main() {
	// Initialize database
	if err := initDB(); err != nil {
		log.Fatal("Database initialization failed:", err)
	}
	defer db.Close()
	
	// Setup routes
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/health", healthHandler)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}