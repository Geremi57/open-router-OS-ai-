package main

	import ("log"
    "os"
    "bufio"
    "fmt"
    "context"
    "github.com/openai/openai-go/v2"
    "github.com/openai/openai-go/v2/option"
		)

func main() {
    scanner := bufio.NewScanner(os.Stdin)
	apikey := os.Getenv("OPENROUTER_API_KEY")
	if apikey == ""{
	log.Fatal("api key is required")
	}
	url := "https://openrouter.ai/api/v1"

client := openai.NewClient(
    option.WithBaseURL(url),
    option.WithAPIKey(apikey),
    )

ctx := context.Background()

messages := []openai.ChatCompletionMessageParamUnion{}

model := "deepseek/deepseek-r1-0528:free"
fmt.Println("whats your question?")
scanner.Scan()
name := scanner.Text()
messages = append(messages, openai.UserMessage(name))


params := openai.ChatCompletionNewParams {
    Model : model,
    Messages : messages,
}
res,err := client.Chat.Completions.New(ctx, params)

if err != nil {
    log.Fatal(err)
}

fmt.Println(res.Choices[0].Message.Content)
}

