package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/your-org/prd-generator/backend/internal/domain"
)

var markdownKeyPattern = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// RenderPRDMarkdown renders the structured PRD document as portable Markdown.
// Unknown JSON shapes are also exported so manually edited sections are retained.
func RenderPRDMarkdown(prd *domain.PRD) ([]byte, error) {
	var document domain.PRDDocument
	if err := json.Unmarshal(prd.Content, &document); err != nil {
		return nil, fmt.Errorf("decode PRD content: %w", err)
	}
	var out strings.Builder
	out.WriteString("# ")
	out.WriteString(strings.TrimSpace(prd.Title))
	out.WriteString("\n\n")
	out.WriteString(fmt.Sprintf("**Status:** %s  \n**Version:** %d  \n**Last updated:** %s\n\n", prd.Status, prd.Version, prd.UpdatedAt.UTC().Format("2006-01-02 15:04 UTC")))

	writeText := func(title, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		out.WriteString("## " + title + "\n\n" + strings.TrimSpace(value) + "\n\n")
	}
	writeList := func(title string, values []string) {
		if len(values) == 0 {
			return
		}
		out.WriteString("## " + title + "\n\n")
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				out.WriteString("- " + strings.TrimSpace(value) + "\n")
			}
		}
		out.WriteString("\n")
	}
	writeRequirements := func(title string, requirements []domain.Requirement) {
		if len(requirements) == 0 {
			return
		}
		out.WriteString("## " + title + "\n\n")
		for _, requirement := range requirements {
			id := strings.TrimSpace(requirement.ID)
			if id == "" {
				id = "Requirement"
			}
			out.WriteString("### " + id + " — " + strings.TrimSpace(requirement.Description) + "\n\n")
			if requirement.Priority != "" {
				out.WriteString("**Priority:** " + requirement.Priority + "\n\n")
			}
			if len(requirement.AcceptanceCriteria) > 0 {
				out.WriteString("**Acceptance criteria**\n\n")
				for _, criterion := range requirement.AcceptanceCriteria {
					out.WriteString("- [ ] " + strings.TrimSpace(criterion) + "\n")
				}
				out.WriteString("\n")
			}
		}
	}

	writeText("Executive Summary", document.ExecutiveSummary)
	writeText("Product Overview", document.ProductOverview)
	writeText("Problem Statement", document.ProblemStatement)
	writeList("Product Vision and Goals", document.VisionAndGoals)
	writeList("Target Audience and User Personas", document.TargetAudience)
	writeList("User Problems and Use Cases", document.UseCases)
	writeRequirements("User Stories", document.UserStories)
	writeRequirements("Functional Requirements", document.FunctionalRequirements)
	writeRequirements("Non-Functional Requirements", document.NonFunctionalRequirements)
	writeList("Feature Specifications", document.FeatureSpecifications)
	writeList("User Journey", document.UserJourney)
	writeList("Scope", document.Scope)
	writeList("Out of Scope", document.OutOfScope)
	writeList("Business Rules", document.BusinessRules)
	writeList("Data Requirements", document.DataRequirements)
	writeList("API Requirements", document.APIRequirements)
	writeText("High-Level Technical Architecture", document.Architecture)
	writeList("Security and Privacy Considerations", document.SecurityAndPrivacy)
	writeList("Analytics and Success Metrics", document.SuccessMetrics)
	writeList("Dependencies and Constraints", document.DependenciesAndConstraints)
	writeList("Risks and Mitigation", document.Risks)
	writeList("Milestones and Development Roadmap", document.Milestones)
	writeList("Open Questions", document.OpenQuestions)
	writeList("Assumptions", document.Assumptions)

	if strings.TrimSpace(out.String()) == "" || !hasDocumentContent(document) {
		var generic any
		if err := json.Unmarshal(prd.Content, &generic); err != nil {
			return nil, fmt.Errorf("decode PRD content: %w", err)
		}
		var fallback bytes.Buffer
		if err := writeGenericMarkdown(&fallback, generic); err != nil {
			return nil, err
		}
		if fallback.Len() > 0 {
			out.WriteString("## Content\n\n")
			out.Write(fallback.Bytes())
		}
	}
	return []byte(out.String()), nil
}

func hasDocumentContent(d domain.PRDDocument) bool {
	return d.ExecutiveSummary != "" || d.ProductOverview != "" || d.ProblemStatement != "" || len(d.VisionAndGoals)+len(d.TargetAudience)+len(d.UseCases)+len(d.UserStories)+len(d.FunctionalRequirements)+len(d.NonFunctionalRequirements)+len(d.FeatureSpecifications)+len(d.UserJourney)+len(d.Scope)+len(d.OutOfScope)+len(d.BusinessRules)+len(d.DataRequirements)+len(d.APIRequirements)+len(d.SecurityAndPrivacy)+len(d.SuccessMetrics)+len(d.DependenciesAndConstraints)+len(d.Risks)+len(d.Milestones)+len(d.OpenQuestions)+len(d.Assumptions) > 0 || d.Architecture != ""
}

func writeGenericMarkdown(out *bytes.Buffer, value any) error {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			item := v[key]
			name := markdownKeyPattern.ReplaceAllString(key, "$1 $2")
			name = strings.NewReplacer("_", " ", "-", " ").Replace(name)
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			out.WriteString("### " + strings.ToUpper(name[:1]) + name[1:] + "\n\n")
			if err := writeGenericMarkdown(out, item); err != nil {
				return err
			}
			out.WriteString("\n")
		}
	case []any:
		for _, item := range v {
			if item == nil {
				continue
			}
			if object, ok := item.(map[string]any); ok {
				label := "Item"
				for _, key := range []string{"title", "name", "id"} {
					if text, ok := object[key].(string); ok && text != "" {
						label = text
						break
					}
				}
				out.WriteString("#### " + label + "\n\n")
				if err := writeGenericMarkdown(out, object); err != nil {
					return err
				}
			} else {
				out.WriteString("- " + fmt.Sprint(item) + "\n")
			}
		}
	case string:
		if strings.TrimSpace(v) != "" {
			out.WriteString(strings.TrimSpace(v) + "\n\n")
		}
	case float64, bool, json.Number:
		out.WriteString(fmt.Sprint(v) + "\n\n")
	case nil:
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return err
		}
		out.WriteString("```json\n" + string(encoded) + "\n```\n\n")
	}
	return nil
}
