package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"
)

// AIService membungkus interaksi dengan Gemini API.
//
// Semua detail SDK (client creation, config, tools, response parsing)
// disembunyikan di sini. Caller hanya perlu tahu satu method: Ask().
// Jika di masa depan kita ganti provider AI, hanya file ini yang berubah.
// defaultTimeout adalah batas waktu maksimum untuk setiap API call ke Gemini.
// 30 detik cukup untuk generasi teks biasa; mencegah goroutine hang selamanya
// saat Gemini API lambat atau rate-limited.
const defaultTimeout = 60 * time.Second

type AIService struct {
	client  *genai.Client
	model   string
	config  *genai.GenerateContentConfig
	timeout time.Duration
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
		client:  client,
		model:   model,
		config:  config,
		timeout: defaultTimeout,
	}, nil
}

// Ask mengirim prompt ke Gemini dan mengembalikan teks respons.
//
// Caller tidak perlu tahu tentang Content/Parts structure, candidate parsing,
// atau error handling internal. Jika tidak ada respons, kembalikan pesan fallback
// daripada error kosong — ini mengikuti prinsip "define errors out of existence".
func (ai *AIService) Ask(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ai.timeout)
	defer cancel()

	contents := []*genai.Content{
		{
			Role:  "user",
			Parts: []*genai.Part{{Text: prompt}},
		},
	}

	resp, err := ai.client.Models.GenerateContent(
		ctx,
		ai.model,
		contents,
		ai.config,
	)
	if err != nil {
		return "", fmt.Errorf("gemini API error: %w", err)
	}

	return extractTextFromResponse(resp), nil
}

// AnalyzeImage mengirim prompt dan data gambar ke Gemini.
// Berguna untuk fitur Vision: "ini foto apa?", OCR, atau estimasi kalori.
func (ai *AIService) AnalyzeImage(prompt string, imageData []byte, mimeType string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ai.timeout)
	defer cancel()

	contents := []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				{
					InlineData: &genai.Blob{
						MIMEType: mimeType,
						Data:     imageData,
					},
				},
				{Text: prompt},
			},
		},
	}

	resp, err := ai.client.Models.GenerateContent(
		ctx,
		ai.model,
		contents,
		ai.config,
	)
	if err != nil {
		return "", fmt.Errorf("gemini vision error: %w", err)
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
		// Filter 1: Lewati "pikiran" internal model (Thought).
		// Gemini 2.0+ sering mengirimkan proses berpikir sebagai part terpisah.
		if part.Thought {
			continue
		}

		// Filter 2: Pastikan part adalah teks murni dan bukan metadata tool.
		// Sesuai dokumentasi SDK, hanya satu field yang boleh terisi dalam satu Part.
		// Jadi kita hanya ambil part yang punya .Text dan bukan hasil/panggilan tool.
		if part.Text != "" && 
			part.FunctionCall == nil && 
			part.ExecutableCode == nil && 
			part.FunctionResponse == nil && 
			part.CodeExecutionResult == nil {
			parts = append(parts, part.Text)
		}
	}

	if len(parts) == 0 {
		return "Hmm, aku gak dapet jawaban nih. Coba tanya lagi ya!"
	}

	return strings.Join(parts, "")
}
