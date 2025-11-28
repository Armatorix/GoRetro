package chatcompletion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Armatorix/GoRetro/internal/models"
)

// Service handles chat completion API calls
type Service struct {
	endpoint string
	apiKey   string
	model    string
	client   *http.Client
}

// NewService creates a new chat completion service
func NewService(endpoint, apiKey, model string) *Service {
	return &Service{
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsConfigured returns true if the service has valid configuration
func (s *Service) IsConfigured() bool {
	return s.endpoint != "" && s.apiKey != "" && s.model != ""
}

// ChatCompletionRequest represents the request to the chat API
type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse represents the response from the chat API
type ChatCompletionResponse struct {
	Choices []Choice `json:"choices"`
}

// Choice represents a completion choice
type Choice struct {
	Message Message `json:"message"`
}

// MergeGroup represents a group of tickets that should be merged
type MergeGroup struct {
	ParentTicketID string   `json:"parent_ticket_id"`
	ChildTicketIDs []string `json:"child_ticket_ids"`
	Reason         string   `json:"reason"`
}

// AutoMergeResponse represents the AI's suggested ticket merges
type AutoMergeResponse struct {
	MergeGroups []MergeGroup `json:"merge_groups"`
}

// SuggestMerges uses AI to suggest which tickets should be merged together
func (s *Service) SuggestMerges(tickets map[string]*models.Ticket) (*AutoMergeResponse, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("chat completion service not configured")
	}

	// Build the prompt with ticket information
	prompt := s.buildMergePrompt(tickets)

	// Create the chat completion request
	reqBody := ChatCompletionRequest{
		Model: s.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an AI assistant helping to group similar retrospective tickets. Analyze the tickets and suggest which ones should be merged together based on their content similarity. Return your response as a JSON object with a 'merge_groups' array, where each group has 'parent_ticket_id', 'child_ticket_ids', and 'reason' fields.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", s.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	// Parse the AI's JSON response
	var mergeResp AutoMergeResponse
	if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &mergeResp); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &mergeResp, nil
}

// buildMergePrompt creates a prompt for the AI to analyze tickets
func (s *Service) buildMergePrompt(tickets map[string]*models.Ticket) string {
	prompt := "Here are the retrospective tickets that need to be analyzed for potential merging:\n\n"

	for id, ticket := range tickets {
		// Skip tickets that are already merged (have a parent)
		if ticket.DeduplicationTicketID != nil {
			continue
		}
		prompt += fmt.Sprintf("Ticket ID: %s\nContent: %s\n\n", id, ticket.Content)
	}

	prompt += `Please analyze these tickets and suggest which ones should be merged together based on content similarity. 
Group tickets that discuss the same topic or issue. For each group:
1. Select the most representative ticket as the parent_ticket_id
2. List other similar tickets as child_ticket_ids
3. Provide a brief reason for the grouping

Return your response as a JSON object with this structure:
{
  "merge_groups": [
    {
      "parent_ticket_id": "ticket-id",
      "child_ticket_ids": ["ticket-id-1", "ticket-id-2"],
      "reason": "Brief explanation"
    }
  ]
}

Only suggest merges where tickets are clearly related. If no tickets should be merged, return an empty merge_groups array.`

	return prompt
}
