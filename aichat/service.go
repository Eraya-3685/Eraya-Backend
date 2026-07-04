package aichat

import (
	"bytes"
	"context"
	"encoding/json"
	"eraya/product"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type service struct {
	repo        Repo
	geminiKey   string
	groqKey     string
	productSvc  product.Service
	frontendURL string
	httpClient  *http.Client
}

func NewService(repo Repo, geminiKey, groqKey string, productSvc product.Service, frontendURL string) Service {
	// Strip trailing slash if present for cleaner URLs
	frontendURL = strings.TrimSuffix(frontendURL, "/")

	return &service{
		repo:        repo,
		geminiKey:   geminiKey,
		groqKey:     groqKey,
		productSvc:  productSvc,
		frontendURL: frontendURL,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

const systemPrompt = `You are "Eraya Shopping Assistant", a helpful, friendly AI assistant for the Eraya e-commerce store.

RULES:
1. Only answer questions related to the store: products, orders, shipping, returns, sizing, recommendations.
2. If asked something unrelated (politics, coding, etc.), politely redirect.
3. When recommending products, ONLY recommend from the PRODUCT CATALOG provided below.
4. When recommending products, you MUST format each product EXACTLY like this: [PRODUCT:slug|Product Name|Price] 
   Example: [PRODUCT:cool-watch|Cool Watch|800]
5. If the catalog has no matching products, say: "I couldn't find exact matches, but you can browse all products at %s/products"
6. Be concise but warm. Use emojis sparingly.
7. You can respond in both Bangla and English.
8. For order tracking, direct users to their Profile page at %s/profile.
9. Format general text clearly, but products MUST use the [PRODUCT:slug|name|price] syntax.
10. Keep responses under 300 words.`

func (s *service) Chat(ctx context.Context, userID *int64, userMessage string, history []ChatMessage) (string, error) {
	// Build product context from the user's message
	productContext := s.buildProductContext(ctx, userMessage)

	fullSystemPrompt := fmt.Sprintf(systemPrompt, s.frontendURL, s.frontendURL)
	if productContext != "" {
		fullSystemPrompt += "\n\nPRODUCT CATALOG (Real products from our store):\n" + productContext
	}

	// Try Gemini first
	if s.geminiKey != "" {
		reply, err := s.callGemini(ctx, fullSystemPrompt, userMessage, history)
		if err == nil && reply != "" {
			if userID != nil {
				_ = s.repo.SaveMessage(ctx, *userID, "user", userMessage)
				_ = s.repo.SaveMessage(ctx, *userID, "assistant", reply)
			}
			return reply, nil
		}
	}

	// Fallback to Groq
	if s.groqKey != "" {
		reply, err := s.callGroq(ctx, fullSystemPrompt, userMessage, history)
		if err == nil && reply != "" {
			if userID != nil {
				_ = s.repo.SaveMessage(ctx, *userID, "user", userMessage)
				_ = s.repo.SaveMessage(ctx, *userID, "assistant", reply)
			}
			return reply, nil
		}
		return "", fmt.Errorf("both AI providers failed")
	}

	return "", fmt.Errorf("no AI provider configured")
}

func (s *service) GetHistory(ctx context.Context, userID int64, limit int) ([]ChatMessage, error) {
	return s.repo.GetHistory(ctx, userID, limit)
}

// buildProductContext queries the database for products relevant to the user's message.
func (s *service) buildProductContext(ctx context.Context, message string) string {
	msg := strings.ToLower(message)

	// Extract price hints
	var maxPrice float64
	pricePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:under|below|within|max|under)\s*(?:৳|tk|taka|bdt)?\s*(\d+)`),
		regexp.MustCompile(`(?:৳|tk|taka|bdt)\s*(\d+)\s*(?:er|er\s+niche|er\s+moddhe)`),
		regexp.MustCompile(`(\d+)\s*(?:৳|tk|taka|bdt)\s*(?:er|er\s+niche|er\s+moddhe)`),
		regexp.MustCompile(`(\d+)\s*(?:er\s+niche|er\s+moddhe|er\s+kom)`),
	}
	for _, p := range pricePatterns {
		if matches := p.FindStringSubmatch(msg); len(matches) > 1 {
			if v, err := strconv.ParseFloat(matches[1], 64); err == nil {
				maxPrice = v
				break
			}
		}
	}

	// Extract search keywords — remove common stop words
	stopWords := map[string]bool{
		"i": true, "want": true, "to": true, "buy": true, "a": true, "the": true,
		"show": true, "me": true, "find": true, "get": true, "good": true, "best": true,
		"some": true, "any": true, "please": true, "can": true, "you": true, "suggest": true,
		"recommend": true, "need": true, "looking": true, "for": true, "under": true,
		"below": true, "within": true, "around": true, "about": true, "cheap": true,
		"budget": true, "ami": true, "amar": true, "chai": true, "lagbe": true, "kinte": true,
		"dekhao": true, "dekhan": true, "valo": true, "ache": true, "ki": true, "kono": true,
		"ekta": true, "daw": true, "dao": true, "er": true, "niche": true, "moddhe": true,
		"taka": true, "tk": true, "bdt": true, "price": true, "kom": true, "max": true,
	}

	words := strings.Fields(msg)
	var keywords []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?৳")
		if len(w) > 2 && !stopWords[w] {
			if _, err := strconv.Atoi(w); err != nil {
				keywords = append(keywords, w)
			}
		}
	}

	search := strings.Join(keywords, " ")
	if search == "" && maxPrice == 0 {
		// General query — fetch popular/latest products
		search = ""
	}

	products, _, err := s.productSvc.GetProducts(ctx, 1, 15, search, nil, "popular", 0, maxPrice, false)
	if err != nil || len(products) == 0 {
		// Try without search if specific search returned nothing
		if search != "" {
			products, _, err = s.productSvc.GetProducts(ctx, 1, 10, "", nil, "popular", 0, maxPrice, false)
		}
		if err != nil || len(products) == 0 {
			return ""
		}
	}

	var sb strings.Builder
	for _, p := range products {
		price := p.BasePrice
		if p.DiscountPrice != nil && *p.DiscountPrice > 0 {
			price = *p.DiscountPrice
		}
		sb.WriteString(fmt.Sprintf("Slug: %s | Name: %s | Price: %.0f", p.Slug, p.Name, price))
		if p.DiscountPrice != nil && *p.DiscountPrice > 0 && *p.DiscountPrice < p.BasePrice {
			sb.WriteString(fmt.Sprintf(" (was %.0f)", p.BasePrice))
		}
		if p.StockCount <= 0 {
			sb.WriteString(" [OUT OF STOCK]")
		}
		if len(p.Colors) > 0 {
			sb.WriteString(fmt.Sprintf(" | Colors: %s", strings.Join(p.Colors, ", ")))
		}
		if len(p.Sizes) > 0 {
			sb.WriteString(fmt.Sprintf(" | Sizes: %s", strings.Join(p.Sizes, ", ")))
		}
		sb.WriteString(fmt.Sprintf(" | Rating: %.1f/5\n", p.AverageRating))
	}
	return sb.String()
}

// ---- Gemini API ----

type geminiRequest struct {
	Contents          []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	GenerationConfig  map[string]interface{} `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *service) callGemini(ctx context.Context, systemPrompt, userMessage string, history []ChatMessage) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", s.geminiKey)

	var contents []geminiContent

	// Add conversation history
	for _, h := range history {
		role := "user"
		if h.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: h.Content}},
		})
	}

	// Add current user message
	contents = append(contents, geminiContent{
		Role:  "user",
		Parts: []geminiPart{{Text: userMessage}},
	})

	reqBody := geminiRequest{
		Contents: contents,
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		GenerationConfig: map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 1024,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("gemini returned status %d", resp.StatusCode)
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", err
	}

	if geminiResp.Error != nil {
		return "", fmt.Errorf("gemini error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("empty gemini response")
}

// ---- Groq API (OpenAI-compatible) ----

type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *service) callGroq(ctx context.Context, systemPrompt, userMessage string, history []ChatMessage) (string, error) {
	url := "https://api.groq.com/openai/v1/chat/completions"

	messages := []groqMessage{
		{Role: "system", Content: systemPrompt},
	}

	for _, h := range history {
		messages = append(messages, groqMessage{Role: h.Role, Content: h.Content})
	}

	messages = append(messages, groqMessage{Role: "user", Content: userMessage})

	reqBody := groqRequest{
		Model:       "llama-3.3-70b-versatile",
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   1024,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.groqKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("groq returned status %d", resp.StatusCode)
	}

	var groqResp groqResponse
	if err := json.Unmarshal(respBody, &groqResp); err != nil {
		return "", err
	}

	if groqResp.Error != nil {
		return "", fmt.Errorf("groq error: %s", groqResp.Error.Message)
	}

	if len(groqResp.Choices) > 0 {
		return groqResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("empty groq response")
}
