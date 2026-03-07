package main

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// AIService membungkus interaksi dengan Gemini API.
//
// Semua detail SDK (client creation, config, tools, response parsing)
// disembunyikan di sini. Caller hanya perlu tahu satu method: Ask().
// Jika di masa depan kita ganti provider AI, hanya file ini yang berubah.
type AIService struct {
	client *genai.Client
	model  string
	config *genai.GenerateContentConfig
}

// NewAIService membuat AIService baru yang siap pakai.
// Menanggung semua kompleksitas: setup client, system instruction, dan Google Search tool.
func NewAIService(apiKey, model, systemPrompt string) (*AIService, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi Gemini client: %w", err)
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: systemPrompt}},
		},
		Tools: []*genai.Tool{
			{GoogleSearch: &genai.GoogleSearch{}},
		},
	}

	return &AIService{
		client: client,
		model:  model,
		config: config,
	}, nil
}

// Ask mengirim prompt ke Gemini dan mengembalikan teks respons.
//
// Caller tidak perlu tahu tentang Content/Parts structure, candidate parsing,
// atau error handling internal. Jika tidak ada respons, kembalikan pesan fallback
// daripada error kosong — ini mengikuti prinsip "define errors out of existence".
func (ai *AIService) Ask(prompt string) (string, error) {
	contents := []*genai.Content{
		{
			Role:  "user",
			Parts: []*genai.Part{{Text: prompt}},
		},
	}

	resp, err := ai.client.Models.GenerateContent(
		context.Background(),
		ai.model,
		contents,
		ai.config,
	)
	if err != nil {
		return "", fmt.Errorf("gemini API error: %w", err)
	}

	return extractTextFromResponse(resp), nil
}

// extractTextFromResponse mengekstrak teks dari response struct Gemini.
// Fungsi private — detail internal tentang bagaimana SDK menyimpan respons.
func extractTextFromResponse(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 {
		return "Hmm, aku gak dapet jawaban nih. Coba tanya lagi ya!"
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "Hmm, aku gak dapet jawaban nih. Coba tanya lagi ya!"
	}

	var parts []string
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			parts = append(parts, part.Text)
		}
	}

	if len(parts) == 0 {
		return "Hmm, aku gak dapet jawaban nih. Coba tanya lagi ya!"
	}

	return strings.Join(parts, "")
}
