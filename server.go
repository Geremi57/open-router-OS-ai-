package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	// "bufio"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

type Request struct {
    Template string `json:"template"`
    Input string `json:"input"`
}

type Response struct {
    Output string `json:"output"`
}

func main() {
    // scanner := bufio.NewScanner(os.Stdin)
	apikey := os.Getenv("OPENROUTER_API_KEY")
	if apikey == ""{
	log.Fatal("api key is required")
	}
	url := "https://openrouter.ai/api/v1"

client := openai.NewClient(
    option.WithBaseURL(url),
    option.WithAPIKey(apikey),
    )
    
        spaceSystemPrompt := `
        Your name is space
    You are Space, a calm and supportive AI.
    You help users reflect and understand their thoughts.
    You do not give medical, legal, or financial advice.
    You speak gently and clearly.
    You never judge the user.
    You ask at most one thoughtful question at the end.
    `
    
    // your existing handler logic here
    
    
    
    http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
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
    
        json.NewDecoder(r.Body).Decode(&req)

        messages := []openai.ChatCompletionMessageParamUnion{
            openai.SystemMessage(spaceSystemPrompt),
            openai.UserMessage(req.Input),
        }
        model := "meta-llama/llama-3.1-8b-instruct"
        // scanner.Scan()
        // name := scanner.Text()
        // messages = append(messages, openai.UserMessage(name))

        params := openai.ChatCompletionNewParams {
            Model : model,
            Messages : messages,
        }

        res,err := client.Chat.Completions.New(context.Background(), params)
        
        if err != nil {
            http.Error(w, err.Error(), 500)
        }

        json.NewEncoder(w).Encode(map[string]string{
    "reply": res.Choices[0].Message.Content,
})

        fmt.Println("what the question")
})
    

// http.HandleFunc("/chat", handler)
    log.Println("Server running on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))

}

