package ai

import (
	"strings"

	"github.com/your-org/prd-generator/backend/internal/config"
	"github.com/your-org/prd-generator/backend/internal/domain"
)

func GetProvidersCatalog() []domain.AIProviderInfo {
	return []domain.AIProviderInfo{
		{
			ID:             "openai",
			Name:           "OpenAI",
			Description:    "Model resmi OpenAI ChatGPT dengan penalaran cerdas dan instruksi terstruktur.",
			DefaultBaseURL: "https://api.openai.com/v1",
			RequiresAPIKey: true,
			SupportsCustom: false,
			DefaultModel:   "gpt-4o-mini",
			Models: []domain.AIModelInfo{
				{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Description: "Cepat, hemat biaya, dan sangat stabil untuk PRD standar (Rekomendasi)", IsDefault: true},
				{ID: "gpt-4o", Name: "GPT-4o", Description: "Model unggulan dengan kemampuan logika arsitektur & penalaran tingkat lanjut", IsDefault: false},
				{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Description: "Performa tinggi untuk dokumen berukuran besar", IsDefault: false},
				{ID: "o3-mini", Name: "o3-mini", Description: "Model penalaran mendalam terbaru OpenAI untuk pemecahan masalah teknis", IsDefault: false},
				{ID: "o1-mini", Name: "o1-mini", Description: "Penalaran matematika, algoritma, dan logika kode terstruktur", IsDefault: false},
			},
		},
		{
			ID:             "gemini",
			Name:           "Google Gemini",
			Description:    "Model AI generasi terbaru dari Google dengan kecepatan tinggi dan jendela konteks luas.",
			DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
			RequiresAPIKey: true,
			SupportsCustom: false,
			DefaultModel:   "gemini-2.0-flash",
			Models: []domain.AIModelInfo{
				{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", Description: "Generasi terbaru Google: respons instan, cerdas & efisien (Rekomendasi)", IsDefault: true},
				{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", Description: "Cepat dan ringan dengan jendela konteks yang luas", IsDefault: false},
				{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", Description: "Penalaran multimodal & analisis dokumen panjang mendalam", IsDefault: false},
			},
		},
		{
			ID:             "groq",
			Name:           "Groq Cloud",
			Description:    "Inference ultra-cepat ditenagai oleh Groq LPU (Language Processing Unit).",
			DefaultBaseURL: "https://api.groq.com/openai/v1",
			RequiresAPIKey: true,
			SupportsCustom: false,
			DefaultModel:   "llama-3.3-70b-versatile",
			Models: []domain.AIModelInfo{
				{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B (Versatile)", Description: "Model Meta Llama 3.3 70B: Sangat cerdas dan seimbang untuk spesifikasi teknis", IsDefault: true},
				{ID: "deepseek-r1-distill-llama-70b", Name: "DeepSeek R1 Distill 70B", Description: "Model reasoning DeepSeek-R1 tercepat via Groq", IsDefault: false},
				{ID: "mixtral-8x7b-32768", Name: "Mixtral 8x7B", Description: "Model MoE Mistral dengan jendela konteks 32k token", IsDefault: false},
			},
		},
		{
			ID:             "deepseek",
			Name:           "DeepSeek",
			Description:    "Model AI dengan efisiensi biaya tertinggi dan logika matematika & kode mumpuni.",
			DefaultBaseURL: "https://api.deepseek.com/v1",
			RequiresAPIKey: true,
			SupportsCustom: false,
			DefaultModel:   "deepseek-chat",
			Models: []domain.AIModelInfo{
				{ID: "deepseek-chat", Name: "DeepSeek V3 (Chat)", Description: "DeepSeek-V3: Model serba bisa dengan akurasi tinggi dan hemat token", IsDefault: true},
				{ID: "deepseek-reasoner", Name: "DeepSeek R1 (Reasoner)", Description: "DeepSeek-R1: Model penalaran berbasis Chain of Thought mendalam", IsDefault: false},
			},
		},
		{
			ID:             "anthropic",
			Name:           "Anthropic Claude",
			Description:    "Kecerdasan bahasa alami terbaik untuk penulisan naratif dokumen dan arsitektur produk.",
			DefaultBaseURL: "https://api.anthropic.com/v1",
			RequiresAPIKey: true,
			SupportsCustom: false,
			DefaultModel:   "claude-3-5-sonnet-20241022",
			Models: []domain.AIModelInfo{
				{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Description: "Kemampuan penulisan dan analisis teknis nomor satu (Rekomendasi)", IsDefault: true},
				{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Description: "Sangat cepat dan responsif untuk draf instan", IsDefault: false},
				{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Description: "Penalaran mendalam untuk masalah sistem skala besar", IsDefault: false},
			},
		},
		{
			ID:             "ollama",
			Name:           "Ollama / Local LLM",
			Description:    "Jalankan model AI secara lokal di komputer atau server internal sendiri (100% privat & gratis).",
			DefaultBaseURL: "http://localhost:11434/v1",
			RequiresAPIKey: false,
			SupportsCustom: true,
			DefaultModel:   "llama3.2",
			Models: []domain.AIModelInfo{
				{ID: "llama3.2", Name: "Llama 3.2", Description: "Model lokal Meta Llama 3.2 (Ringan & Cepat)", IsDefault: true},
				{ID: "deepseek-r1:8b", Name: "DeepSeek R1 8B", Description: "Model reasoning lokal DeepSeek-R1 8B", IsDefault: false},
				{ID: "mistral", Name: "Mistral 7B", Description: "Model lokal Mistral 7B standar industri", IsDefault: false},
				{ID: "qwen2.5-coder", Name: "Qwen 2.5 Coder", Description: "Model lokal spesialis software architecture & koding", IsDefault: false},
			},
		},
		{
			ID:             "custom",
			Name:           "Custom OpenAI-Compatible",
			Description:    "Hubungkan ke server LLM apa pun (vLLM, LM Studio, OpenRouter, Private Gateway).",
			DefaultBaseURL: "http://localhost:8000/v1",
			RequiresAPIKey: false,
			SupportsCustom: true,
			DefaultModel:   "custom-model",
			Models:         []domain.AIModelInfo{},
		},
	}
}

// ResolveSettings calculates the effective provider, baseURL, apiKey, and model
func ResolveSettings(userSettings *domain.UserAISettings, overrideProvider, overrideModel string, defaultCfg config.Config) (provider, baseURL, apiKey, model string) {
	catalog := GetProvidersCatalog()
	catalogMap := make(map[string]domain.AIProviderInfo)
	for _, p := range catalog {
		catalogMap[p.ID] = p
	}

	provider = "openai"
	model = defaultCfg.AIModel
	if model == "" {
		model = "gpt-4o-mini"
	}
	baseURL = defaultCfg.AIBaseURL
	apiKey = defaultCfg.AIAPIKey

	if userSettings != nil && userSettings.Provider != "" {
		provider = userSettings.Provider
		if info, ok := catalogMap[provider]; ok {
			baseURL = info.DefaultBaseURL
			model = info.DefaultModel
		}
		if userSettings.BaseURL != "" {
			baseURL = userSettings.BaseURL
		}
		if userSettings.Model != "" {
			model = userSettings.Model
		}
		if userSettings.CustomModel != "" {
			model = userSettings.CustomModel
		}
		if userSettings.APIKey != "" {
			apiKey = userSettings.APIKey
		}
	}

	// Apply per-request overrides if provided
	if overrideProvider != "" {
		provider = overrideProvider
		if info, ok := catalogMap[provider]; ok {
			baseURL = info.DefaultBaseURL
			model = info.DefaultModel
		}
	}
	if overrideModel != "" {
		model = overrideModel
	}

	baseURL = strings.TrimRight(baseURL, "/")
	return provider, baseURL, apiKey, model
}
