package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type HelixYaml struct {
	Tests []struct {
		Name  string `yaml:"name"`
		Steps []struct {
			Prompt         string `yaml:"prompt"`
			ExpectedOutput string `yaml:"expected_output"`
		} `yaml:"steps"`
	} `yaml:"tests"`
}

type ChatRequest struct {
	Model     string    `json:"model"`
	SessionID string    `json:"session_id"`
	System    string    `json:"system"`
	Messages  []Message `json:"messages"`
	AppID     string    `json:"app_id"`
}

type Message struct {
	Role    string  `json:"role"`
	Content Content `json:"content"`
}

type Content struct {
	ContentType string   `json:"content_type"`
	Parts       []string `json:"parts"`
}

type TestResult struct {
	TestName  string `json:"test_name"`
	Prompt    string `json:"prompt"`
	Response  string `json:"response"`
	Expected  string `json:"expected"`
	Result    string `json:"result"`
	Reason    string `json:"reason"`
	SessionID string `json:"session_id"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func main() {
	// Read helix.yaml
	yamlFile, err := ioutil.ReadFile("helix.yaml")
	if err != nil {
		fmt.Printf("Error reading helix.yaml: %v\n", err)
		return
	}

	var helixYaml HelixYaml
	err = yaml.Unmarshal(yamlFile, &helixYaml)
	if err != nil {
		fmt.Printf("Error parsing helix.yaml: %v\n", err)
		return
	}

	// Get app_id from environment variable
	appID := os.Getenv("HELIX_APP_ID")
	if appID == "" {
		fmt.Println("Error: HELIX_APP_ID environment variable not set")
		return
	}

	// Get API key from environment variable
	apiKey := os.Getenv("HELIX_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: HELIX_API_KEY environment variable not set")
		return
	}

	var results []TestResult

	// Get HELIX_URL from environment variable
	helixURL := os.Getenv("HELIX_URL")
	if helixURL == "" {
		fmt.Println("Error: HELIX_URL environment variable not set")
		return
	}

	// Iterate over tests
	for _, test := range helixYaml.Tests {
		for _, step := range test.Steps {
			// Send test API request
			chatReq := ChatRequest{
				Messages: []Message{
					{
						Role: "user",
						Content: Content{
							ContentType: "text",
							Parts:       []string{step.Prompt},
						},
					},
				},
				AppID: appID,
			}

			jsonData, err := json.Marshal(chatReq)
			if err != nil {
				fmt.Printf("Error marshaling JSON: %v\n", err)
				continue
			}

			req, err := http.NewRequest("POST", helixURL+"/api/v1/sessions/chat", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Printf("Error creating request: %v\n", err)
				continue
			}

			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error sending request: %v\n", err)
				continue
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response: %v\n", err)
				continue
			}

			// Parse the response
			var chatResp ChatResponse
			err = json.Unmarshal(body, &chatResp)
			if err != nil {
				fmt.Printf("Error parsing response JSON: %v\n", err)
				continue
			}

			if len(chatResp.Choices) == 0 {
				fmt.Println("Error: No choices in the response")
				continue
			}

			responseContent := chatResp.Choices[0].Message.Content
			fmt.Printf("Chat response content: %s\n", responseContent)

			// Evaluate response using another LLM call
			evalReq := ChatRequest{
				Model:  "llama3.1:8b-instruct-q8_0",
				System: "You are an AI assistant tasked with evaluating test results. Output only PASS or FAIL followed by a brief explanation on the next line.",
				Messages: []Message{
					{
						Role: "user",
						Content: Content{
							ContentType: "text",
							Parts:       []string{fmt.Sprintf("Does this response:\n\n%s\n\nsatisfy the expected output:\n\n%s", responseContent, step.ExpectedOutput)},
						},
					},
				},
			}

			jsonData, err = json.Marshal(evalReq)
			if err != nil {
				fmt.Printf("Error marshaling JSON for evaluation: %v\n", err)
				continue
			}

			req, err = http.NewRequest("POST", helixURL+"/api/v1/sessions/chat", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Printf("Error creating evaluation request: %v\n", err)
				continue
			}

			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("Content-Type", "application/json")

			resp, err = client.Do(req)
			if err != nil {
				fmt.Printf("Error sending evaluation request: %v\n", err)
				continue
			}
			defer resp.Body.Close()

			evalBody, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading evaluation response: %v\n", err)
				continue
			}

			// Parse the evaluation response
			var evalResp ChatResponse
			err = json.Unmarshal(evalBody, &evalResp)
			if err != nil {
				fmt.Printf("Error parsing evaluation response JSON: %v\n", err)
				continue
			}

			if len(evalResp.Choices) == 0 {
				fmt.Println("Error: No choices in the evaluation response")
				continue
			}

			evalContent := evalResp.Choices[0].Message.Content
			fmt.Printf("Evaluation response content: %s\n", evalContent)

			result := TestResult{
				TestName:  test.Name,
				Prompt:    step.Prompt,
				Response:  responseContent,
				Expected:  step.ExpectedOutput,
				Result:    evalContent[:4], // Assuming PASS or FAIL
				Reason:    evalContent[5:], // Explanation after PASS or FAIL
				SessionID: chatResp.ID,     // Use the ID field for SessionID
			}

			results = append(results, result)
		}
	}

	// Display results in a table with link
	fmt.Println("Test Results:")
	fmt.Println("--------------------------------------------------------------------------------------------------------------------")
	fmt.Printf("%-20s | %-10s | %-10s | %s\n", "Test Name", "Result", "Session ID", "Link")
	fmt.Println("--------------------------------------------------------------------------------------------------------------------")
	for _, result := range results {
		link := fmt.Sprintf("https://app.tryhelix.ai/dashboard?tab=llm_calls&filter_sessions=%s", result.SessionID)
		fmt.Printf("%-20s | %-10s | %-10s | %s\n", result.TestName, result.Result, result.SessionID, link)
	}

	// Write results to JSON file
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("results_%s.json", timestamp)
	jsonResults, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling results to JSON: %v\n", err)
		return
	}

	err = os.WriteFile(filename, jsonResults, 0644)
	if err != nil {
		fmt.Printf("Error writing results to file: %v\n", err)
		return
	}

	fmt.Printf("\nResults written to %s\n", filename)
}
