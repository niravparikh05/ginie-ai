package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/niravparikh05/ginie-ai/terraform"
)

const (
	work_dir = "gen-ai-tf"
)

const (
	apiEndpoint = "https://api.openai.com/v1"
)

var (
	messages          []azopenai.ChatRequestMessageClassification
	modelDeploymentID string
	temperature       float32
)

func main() {
	fmt.Println("Hey There ! I am Ginie, What would you like to spin up today ?")

	if len(os.Getenv("OPENAI_API_KEY")) == 0 || len(os.Getenv("OPENAI_MODEL")) == 0 {
		fmt.Fprintf(os.Stderr, "Skipping example, environment variables missing\n")
		return
	}

	keyCredential := azcore.NewKeyCredential(os.Getenv("OPENAI_API_KEY"))
	client, err := azopenai.NewClientForOpenAI(apiEndpoint, keyCredential, nil)
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	modelDeploymentID = os.Getenv("OPENAI_MODEL")
	temperature = float32(0.8)

	/// This is a conversation in progress.
	// NOTE: all messages, regardless of role, count against token usage for this API.
	messages = []azopenai.ChatRequestMessageClassification{
		// You set the tone and rules of the conversation with a prompt as the system role.
		&azopenai.ChatRequestSystemMessage{Content: to.Ptr(`You are Ginie, an AI conversation assistant that builds and deploys Cloud Infrastructure written in Terraform.
		Generate a description of the Terraform program you will define, followed by a single Terraform program which includes default values in response to each of my Instructions.
		I will then deploy that program for you and let you know if there were errors.
		You should modify the current program based on my instructions.
		You should not start from scratch unless asked.`)},

		// The user asks a question
		&azopenai.ChatRequestUserMessage{Content: azopenai.NewChatRequestUserMessageContent("Can you help create a working terraform template with default values and credentials section which I will update later if needed?")},

		// The reply would come back from the ChatGPT. You'd add it to the conversation so we can maintain context.
		&azopenai.ChatRequestAssistantMessage{Content: to.Ptr("Of course! Which resource would you like to create?")},
	}

	_, err = client.GetChatCompletions(context.Background(), azopenai.ChatCompletionsOptions{
		// This is a conversation in progress.
		// NOTE: all messages count against token usage for this API.
		Messages:       messages,
		DeploymentName: &modelDeploymentID,
		Temperature:    &temperature,
	}, nil)

	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(">>> ")
		scanner.Scan()
		err := scanner.Err()
		if err != nil {
			log.Fatal(err)
		}
		query := scanner.Text()
		//query := "!deploy"
		switch query {
		case "!quit":
			return
		case "!deploy":

			response, err := callLlm(client, "respond with only terraform hcl code")
			if err != nil {
				log.Fatalf("ERROR: %s", err)
			}
			writeToFile("main.tf", response)

			// deploy using terraform
			fmt.Println("hold on ! publishing the infrastructure for you.")
			config := terraform.NewDriverConfig([]string{"init", "plan", "apply"}, "1.5.7", work_dir)
			tfRunner := terraform.NewTerraformRunner(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})), config)
			if err := tfRunner.Execute(); err != nil {
				fmt.Println("failed to publish infrastructure: ", err.Error())
			}
		case "!destroy":
			// destroy using terraform
			fmt.Println("hold on ! destroying the infrastructure for you.")
			config := terraform.NewDriverConfig([]string{"init", "destroy"}, "1.5.7", string(work_dir))
			tfRunner := terraform.NewTerraformRunner(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})), config)
			if err := tfRunner.Execute(); err != nil {
				fmt.Println("failed to destroy infrastructure: ", err.Error())
			}
		default:
			response, err := callLlm(client, query)
			if err != nil {
				log.Fatalf("ERROR: %s", err)
			}
			fmt.Fprintf(os.Stderr, "%s\n", response)
		}
	}

}

func callLlm(client *azopenai.Client, query string) (string, error) {
	messages = append(messages, &azopenai.ChatRequestUserMessage{
		Content: azopenai.NewChatRequestUserMessageContent(query),
	})

	resp, err := client.GetChatCompletions(context.Background(), azopenai.ChatCompletionsOptions{
		// This is a conversation in progress.
		// NOTE: all messages count against token usage for this API.
		Messages:       messages,
		DeploymentName: &modelDeploymentID,
	}, nil)

	if err != nil {
		log.Fatalf("ERROR: %s", err)
		return "", err
	}

	completion := ""
	for _, choice := range resp.Choices {

		if choice.ContentFilterResults != nil {
			fmt.Fprintf(os.Stderr, "Content filter results\n")

			if choice.ContentFilterResults.Error != nil {
				fmt.Fprintf(os.Stderr, "  Error:%v\n", choice.ContentFilterResults.Error)
			}

			fmt.Fprintf(os.Stderr, "  Hate: sev: %v, filtered: %v\n", *choice.ContentFilterResults.Hate.Severity, *choice.ContentFilterResults.Hate.Filtered)
			fmt.Fprintf(os.Stderr, "  SelfHarm: sev: %v, filtered: %v\n", *choice.ContentFilterResults.SelfHarm.Severity, *choice.ContentFilterResults.SelfHarm.Filtered)
			fmt.Fprintf(os.Stderr, "  Sexual: sev: %v, filtered: %v\n", *choice.ContentFilterResults.Sexual.Severity, *choice.ContentFilterResults.Sexual.Filtered)
			fmt.Fprintf(os.Stderr, "  Violence: sev: %v, filtered: %v\n", *choice.ContentFilterResults.Violence.Severity, *choice.ContentFilterResults.Violence.Filtered)
		}

		if choice.Message != nil && choice.Message.Content != nil {
			completion = *choice.Message.Content
		}

	}
	return completion, nil
}

func writeToFile(fileName, content string) {
	f, err := os.Create(fmt.Sprintf("%s/%s", work_dir, fileName))
	if err != nil {
		log.Fatal(err)
	}

	var hcl string
	strs := strings.SplitAfter(content, "```")
	hcl = strings.ReplaceAll(strs[1], "```", "")
	hcl = strings.ReplaceAll(hcl, "hcl", "")

	_, err = f.WriteString(strings.TrimSpace(hcl))
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
}
