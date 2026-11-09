package service

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/prd-generator/backend/internal/domain"
)

func TestRenderPRDMarkdownIncludesRequirementsAndCriteria(t *testing.T) {
	doc := domain.PRDDocument{ExecutiveSummary: "A product summary", FunctionalRequirements: []domain.Requirement{{ID: "FR-001", Description: "User can sign in", Priority: "P0", AcceptanceCriteria: []string{"Valid credentials create a session"}}}}
	content, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	markdown, err := RenderPRDMarkdown(&domain.PRD{Title: "Sample PRD", Status: "draft", Version: 2, Content: content, UpdatedAt: time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Sample PRD", "## Executive Summary", "### FR-001 — User can sign in", "**Priority:** P0", "- [ ] Valid credentials create a session"} {
		if !strings.Contains(string(markdown), expected) {
			t.Errorf("markdown missing %q", expected)
		}
	}
}

func TestExportFormats(t *testing.T) {
	markdown := "# Product\n\n## Summary\n\nReadable content\n"
	pdf, err := RenderPRDPDF(markdown)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatal("PDF signature missing")
	}
	docx, err := RenderPRDDOCX(markdown)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatalf("invalid DOCX archive: %v", err)
	}
	found := false
	for _, file := range archive.File {
		if file.Name == "word/document.xml" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("DOCX document.xml missing")
	}

	doc := domain.PRDDocument{
		ExecutiveSummary: "Test Summary",
		UserStories: []domain.Requirement{
			{ID: "US-001", Description: "As a user, I want to login", Priority: "P0", AcceptanceCriteria: []string{"Given valid credentials, login succeeds"}},
		},
		FunctionalRequirements: []domain.Requirement{
			{ID: "FR-001", Description: "Support OAuth2", Priority: "P1", AcceptanceCriteria: []string{"Google provider supported"}},
		},
	}
	content, _ := json.Marshal(doc)
	prd := &domain.PRD{Title: "Export Test", Content: content}

	jiraCSV, err := RenderPRDJiraCSV(prd)
	if err != nil {
		t.Fatalf("RenderPRDJiraCSV error: %v", err)
	}
	if !strings.Contains(string(jiraCSV), "Issue Type,Issue key,Summary,Description,Priority") {
		t.Errorf("jiraCSV missing header")
	}
	if !strings.Contains(string(jiraCSV), "US-001") || !strings.Contains(string(jiraCSV), "Highest") {
		t.Errorf("jiraCSV missing expected data")
	}

	linearCSV, err := RenderPRDLinearCSV(prd)
	if err != nil {
		t.Fatalf("RenderPRDLinearCSV error: %v", err)
	}
	if !strings.Contains(string(linearCSV), "Title,Description,Priority,Estimate") {
		t.Errorf("linearCSV missing header")
	}
	if !strings.Contains(string(linearCSV), "US-001") || !strings.Contains(string(linearCSV), "Urgent") {
		t.Errorf("linearCSV missing expected data")
	}
}

func TestValidatePRDDocumentRequiresUniqueTestableRequirements(t *testing.T) {
	valid := domain.PRDDocument{ExecutiveSummary: "Summary", ProductOverview: "Overview", ProblemStatement: "Problem", FunctionalRequirements: []domain.Requirement{{ID: "FR-001", Description: "Sign in", Priority: "P1", AcceptanceCriteria: []string{"Valid credentials open dashboard"}}}}
	if err := ValidatePRDDocument(valid); err != nil {
		t.Fatalf("valid document rejected: %v", err)
	}
	valid.FunctionalRequirements = append(valid.FunctionalRequirements, valid.FunctionalRequirements[0])
	if err := ValidatePRDDocument(valid); err == nil {
		t.Fatal("duplicate requirement ID accepted")
	}
}
