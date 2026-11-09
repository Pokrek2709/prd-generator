package domain

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
type Project struct {
	ID          string    `json:"id"`
	UserID      string    `json:"-"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
type PRD struct {
	ID        string          `json:"id"`
	UserID    string          `json:"-"`
	ProjectID string          `json:"projectId"`
	Title     string          `json:"title"`
	Status    string          `json:"status"`
	Content   json.RawMessage `json:"content"`
	Version   int             `json:"version"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}
type PRDVersion struct {
	ID        string          `json:"id"`
	PRDID     string          `json:"prdId"`
	Version   int             `json:"version"`
	Title     string          `json:"title"`
	Status    string          `json:"status"`
	Content   json.RawMessage `json:"content"`
	CreatedAt time.Time       `json:"createdAt"`
}
type RefreshToken struct {
	Hash, UserID string
	ExpiresAt    time.Time
}
type Idea struct {
	ProductName    string            `json:"productName"`
	Description    string            `json:"description"`
	Problem        string            `json:"problem"`
	TargetUsers    string            `json:"targetUsers"`
	BusinessGoals  string            `json:"businessGoals"`
	Platform       string            `json:"platform"`
	ProductType    string            `json:"productType"`
	Features       []string          `json:"features"`
	Technology     string            `json:"technology"`
	Constraints    string            `json:"constraints"`
	Language       string            `json:"language"`
	DetailLevel    string            `json:"detailLevel"`
	Clarifications map[string]string `json:"clarifications,omitempty"`
}
type ClarificationQuestion struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}
type ClarificationSession struct {
	ID        string                  `json:"id"`
	Idea      Idea                    `json:"idea"`
	Questions []ClarificationQuestion `json:"questions"`
	Answers   map[string]string       `json:"answers"`
	Ready     bool                    `json:"ready"`
	CreatedAt time.Time               `json:"createdAt"`
	UpdatedAt time.Time               `json:"updatedAt"`
}
type ShareLink struct {
	ID        string    `json:"id"`
	PRDID     string    `json:"prdId"`
	Token     string    `json:"token,omitempty"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
type Requirement struct {
	ID                 string   `json:"id"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	AcceptanceCriteria []string `json:"acceptanceCriteria"`
}
type PRDDocument struct {
	ExecutiveSummary           string        `json:"executiveSummary"`
	ProductOverview            string        `json:"productOverview"`
	ProblemStatement           string        `json:"problemStatement"`
	VisionAndGoals             []string      `json:"visionAndGoals"`
	TargetAudience             []string      `json:"targetAudience"`
	UseCases                   []string      `json:"useCases"`
	UserStories                []Requirement `json:"userStories"`
	FunctionalRequirements     []Requirement `json:"functionalRequirements"`
	NonFunctionalRequirements  []Requirement `json:"nonFunctionalRequirements"`
	FeatureSpecifications      []string      `json:"featureSpecifications"`
	UserJourney                []string      `json:"userJourney"`
	Scope                      []string      `json:"scope"`
	OutOfScope                 []string      `json:"outOfScope"`
	BusinessRules              []string      `json:"businessRules"`
	DataRequirements           []string      `json:"dataRequirements"`
	APIRequirements            []string      `json:"apiRequirements"`
	Architecture               string        `json:"architecture"`
	SecurityAndPrivacy         []string      `json:"securityAndPrivacy"`
	SuccessMetrics             []string      `json:"successMetrics"`
	DependenciesAndConstraints []string      `json:"dependenciesAndConstraints"`
	Risks                      []string      `json:"risks"`
	Milestones                 []string      `json:"milestones"`
	OpenQuestions              []string      `json:"openQuestions"`
	Assumptions                []string      `json:"assumptions"`
}

type UserAISettings struct {
	UserID      string    `json:"userId"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	BaseURL     string    `json:"baseUrl,omitempty"`
	APIKey      string    `json:"apiKey,omitempty"`
	CustomModel string    `json:"customModel,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AIModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsDefault   bool   `json:"isDefault"`
}

type AIProviderInfo struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	DefaultBaseURL string        `json:"defaultBaseUrl"`
	RequiresAPIKey bool          `json:"requiresApiKey"`
	SupportsCustom bool          `json:"supportsCustom"`
	DefaultModel   string        `json:"defaultModel"`
	Models         []AIModelInfo `json:"models"`
}
