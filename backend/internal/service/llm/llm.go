package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/service/topicnorm"
)

// maxErrorBodyBytes bounds how much of an LLM error body we propagate up into
// error messages and logs. Providers occasionally echo request headers back
// on failure, so we cap aggressively to avoid leaking the API key and to
// keep error strings readable.
const maxErrorBodyBytes = 512

func readErrorBody(r io.Reader) string {
	body, _ := io.ReadAll(io.LimitReader(r, maxErrorBodyBytes+1))
	if len(body) > maxErrorBodyBytes {
		return string(body[:maxErrorBodyBytes]) + "...(truncated)"
	}
	return string(body)
}

type Service interface {
	SummarizeArticle(ctx context.Context, title, content string) (*ArticleSummary, error)
	SummarizeArticleWithKey(ctx context.Context, title, content, apiKey string) (*ArticleSummary, error)
	SummarizeArticleWithConfig(ctx context.Context, title, content, provider, apiURL, apiKey, model string) (*ArticleSummary, error)
}

type ArticleSummary struct {
	Summary         string
	KeyPoints       []string
	ImportanceScore int
	Topics          []string
}

type service struct {
	apiURL     string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewService(apiURL, apiKey, model string) Service {
	return &service{
		apiURL: apiURL,
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type anthropicRequest struct {
	Model      string    `json:"model"`
	Messages   []message `json:"messages"`
	MaxTokens  int       `json:"max_tokens"`
	System     string    `json:"system,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// OpenAI-compatible request/response types
type openAIRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type summaryResponse struct {
	Summary         string   `json:"summary"`
	KeyPoints       []string `json:"key_points"`
	ImportanceScore int      `json:"importance_score"`
	Topics          []string `json:"topics"`
}

func (s *service) SummarizeArticle(ctx context.Context, title, content string) (*ArticleSummary, error) {
	return s.summarizeWithKey(ctx, title, content, s.apiKey)
}

func (s *service) SummarizeArticleWithKey(ctx context.Context, title, content, apiKey string) (*ArticleSummary, error) {
	// If no custom key provided, use default
	if apiKey == "" {
		apiKey = s.apiKey
	}
	return s.summarizeWithKey(ctx, title, content, apiKey)
}

func (s *service) summarizeWithKey(ctx context.Context, title, content, apiKey string) (*ArticleSummary, error) {
	prompt := buildSummarizationPrompt(title, content)

	reqBody := anthropicRequest{
		Model:     s.model,
		MaxTokens: 2048,
		System:    "You are a helpful assistant that summarizes articles. Always respond with valid JSON.",
		Messages: []message{
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

	req, err := http.NewRequestWithContext(ctx, "POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	// Extract JSON from markdown code blocks if present
	responseText := anthropicResp.Content[0].Text
	responseText = extractJSON(responseText)

	var summaryResp summaryResponse
	if err := json.Unmarshal([]byte(responseText), &summaryResp); err != nil {
		return nil, fmt.Errorf("failed to parse summary response: %w", err)
	}

	// Normalize and deduplicate topics, then cap to the configured max.
	normalizedTopics := capTopics(normalizeTopics(summaryResp.Topics))

	return &ArticleSummary{
		Summary:         summaryResp.Summary,
		KeyPoints:       summaryResp.KeyPoints,
		ImportanceScore: summaryResp.ImportanceScore,
		Topics:          normalizedTopics,
	}, nil
}

func (s *service) SummarizeArticleWithConfig(ctx context.Context, title, content, provider, apiURL, apiKey, model string) (*ArticleSummary, error) {
	// Use defaults if not provided
	if provider == "" {
		provider = "anthropic"
	}
	if apiKey == "" {
		apiKey = s.apiKey
	}
	if model == "" {
		model = s.model
	}

	prompt := buildSummarizationPrompt(title, content)

	// Determine if using Anthropic or OpenAI format
	isAnthropic := strings.ToLower(provider) == "anthropic"

	var jsonData []byte
	var err error
	var effectiveURL string

	if isAnthropic {
		// Use Anthropic native format
		reqBody := anthropicRequest{
			Model:     model,
			MaxTokens: 2048,
			System:    "You are a helpful assistant that summarizes articles. Always respond with valid JSON.",
			Messages: []message{
				{
					Role:    "user",
					Content: prompt,
				},
			},
		}
		jsonData, err = json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		// Use configured URL or default Anthropic endpoint
		if apiURL != "" {
			effectiveURL = apiURL
		} else {
			effectiveURL = s.apiURL
		}
	} else {
		// Use OpenAI-compatible format
		reqBody := openAIRequest{
			Model:       model,
			MaxTokens:   2048,
			Temperature: 0.3,
			Messages: []message{
				{
					Role:    "system",
					Content: "You are a helpful assistant that summarizes articles. Always respond with valid JSON.",
				},
				{
					Role:    "user",
					Content: prompt,
				},
			},
		}
		jsonData, err = json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		// Use custom URL or default to OpenAI
		if apiURL != "" {
			effectiveURL = apiURL
		} else {
			effectiveURL = "https://api.openai.com/v1/chat/completions"
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", effectiveURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers based on provider
	req.Header.Set("Content-Type", "application/json")
	if isAnthropic {
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var responseText string

	if isAnthropic {
		var anthropicResp anthropicResponse
		if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if len(anthropicResp.Content) == 0 {
			return nil, fmt.Errorf("no content in response")
		}

		responseText = anthropicResp.Content[0].Text
	} else {
		var openAIResp openAIResponse
		if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if openAIResp.Error != nil {
			return nil, fmt.Errorf("LLM API error: %s", openAIResp.Error.Message)
		}

		if len(openAIResp.Choices) == 0 {
			return nil, fmt.Errorf("no choices in response")
		}

		responseText = openAIResp.Choices[0].Message.Content
	}

	// Extract JSON from markdown code blocks if present
	responseText = extractJSON(responseText)

	var summaryResp summaryResponse
	if err := json.Unmarshal([]byte(responseText), &summaryResp); err != nil {
		return nil, fmt.Errorf("failed to parse summary response: %w", err)
	}

	// Normalize and deduplicate topics, then cap to the configured max.
	normalizedTopics := capTopics(normalizeTopics(summaryResp.Topics))

	return &ArticleSummary{
		Summary:         summaryResp.Summary,
		KeyPoints:       summaryResp.KeyPoints,
		ImportanceScore: summaryResp.ImportanceScore,
		Topics:          normalizedTopics,
	}, nil
}

func extractJSON(text string) string {
	// Remove markdown code blocks if present
	// Handles both ```json and ``` variants
	textBytes := bytes.TrimSpace([]byte(text))

	// Check if it starts with ```
	if bytes.HasPrefix(textBytes, []byte("```")) {
		// Find the first newline after ```
		firstNewline := bytes.IndexByte(textBytes, '\n')
		if firstNewline != -1 {
			textBytes = textBytes[firstNewline+1:]
		}

		// Remove trailing ```
		if bytes.HasSuffix(textBytes, []byte("```")) {
			textBytes = textBytes[:len(textBytes)-3]
		}

		textBytes = bytes.TrimSpace(textBytes)
	}

	return string(textBytes)
}

// normalizeTopics delegates to the shared topicnorm package so the LLM and
// the repository apply identical rules.
func normalizeTopics(topics []string) []string {
	return topicnorm.Normalize(topics)
}

// maxTopicsPerArticle caps the topic count stored on an article. The LLM is
// instructed to return at most 2; this is the enforcement side.
const maxTopicsPerArticle = 2

func capTopics(topics []string) []string {
	if len(topics) > maxTopicsPerArticle {
		return topics[:maxTopicsPerArticle]
	}
	return topics
}

func buildSummarizationPrompt(title, content string) string {
	return fmt.Sprintf(`Analyze the following article and provide a JSON response with this exact structure:

{
  "summary": "detailed paragraph summary here",
  "key_points": ["point 1", "point 2", "point 3"],
  "importance_score": 3,
  "topics": ["Go"]
}

Note: The "topics" array should contain 1-2 broad categories maximum.

Summary guidelines:
- Write 2-3 detailed paragraphs (150-250 words total)
- Provide enough context and detail so readers can understand the article without visiting the original
- Include key facts, statistics, examples, and main arguments from the article
- Explain WHY something matters, not just WHAT it is
- Be informative and substantive - this is the main content users will read
- Write in flowing prose, not bullet points

Key points guidelines:
- Keep these concise - single sentence takeaways (3-5 points)
- These should be brief highlights, not detailed explanations
- Focus on actionable insights or memorable facts

Importance score guidelines:
- 5: Must read - breakthrough news, major developments
- 4: High priority - important updates, significant analysis
- 3: Medium interest - worth reading, useful information
- 2: Low priority - minor updates, niche topics
- 1: Likely skip - trivial content, fluff

Topics guidelines - BE VERY STRICT:
- ONLY use 1-2 broad, top-level categories (maximum 2, prefer 1)
- ONLY use these approved categories or very similar broad terms:
  * Languages: Go, Rust, Python, JavaScript, TypeScript, Java, C++
  * Cloud/Infra: Kubernetes, Docker, Cloud, AWS, GCP, Azure
  * Domains: Security, AI, Databases, Web, APIs, DevOps
  * General: Engineering, Architecture, Performance, Testing, Git, Linux
- Use SINGLE WORDS when possible (e.g., "Security" not "Cybersecurity")
- Consolidate specific tools to categories:
  * "PostgreSQL" → "Databases"
  * "Machine Learning" → "AI"
  * "React" → "Web"
  * "REST API" → "APIs"
- NEVER use version numbers, specific features, or multi-word descriptions
- Good: "Go", "Kubernetes", "Security", "AI"
- Bad: "Go 1.21", "Kubernetes Deployment", "Application Security", "LLM Fine-tuning"

Article Title: %s

Article Content:
%s

Respond ONLY with the JSON object, no other text.`, title, content)
}
