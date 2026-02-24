//go:build ignore

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	fmt.Println("Downloading test image from Lorem Picsum...")
	imgResp, err := http.Get("https://picsum.photos/id/237/200/300")
	if err != nil {
		fmt.Printf("Failed to download image: %v\n", err)
		os.Exit(1)
	}
	defer imgResp.Body.Close()

	imgData, err := io.ReadAll(imgResp.Body)
	if err != nil {
		fmt.Printf("Failed to read image data: %v\n", err)
		os.Exit(1)
	}

	b64Image := base64.StdEncoding.EncodeToString(imgData)

	fmt.Println("Sending request to proxy server at http://localhost:8000/v1/chat/completions ...")

	payload := map[string]interface{}{
		"model": "moondream",
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "text", "text": "Describe this image in one short sentence."},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:image/jpeg;base64,%s", b64Image),
						},
					},
				},
			},
		},
		"max_tokens": 50,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Failed to marshal JSON payload: %v\n", err)
		os.Exit(1)
	}

	req, err := http.NewRequest("POST", "http://localhost:8000/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-alice-001")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Request to proxy failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Proxy returned status %d: %s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		fmt.Printf("Failed to parse JSON response: %v\nRaw response: %s\n", err, string(respBody))
		os.Exit(1)
	}

	fmt.Println("\n--- Model Response ---")
	if len(response.Choices) > 0 {
		fmt.Println(response.Choices[0].Message.Content)
	} else {
		fmt.Println("(No matching choices returned)")
	}

	fmt.Println("\n--- Token Usage ---")
	fmt.Printf("Prompt: %d, Completion: %d, Total: %d\n",
		response.Usage.PromptTokens,
		response.Usage.CompletionTokens,
		response.Usage.TotalTokens,
	)
}
