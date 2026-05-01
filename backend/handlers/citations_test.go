package handlers

import (
	"cancer-to-hell/pubmed"
	"strings"
	"testing"
)

func TestValidateCitations_AllAllowed(t *testing.T) {
	allowed := map[string]bool{"123": true, "456": true}
	in := "Per recent evidence (PMID: 123) and (PMID: 456), treatment X is preferred."

	out, hallucinated := validateCitations(in, allowed)

	if out != in {
		t.Errorf("expected text unchanged, got %q", out)
	}
	if len(hallucinated) != 0 {
		t.Errorf("expected no hallucinations, got %v", hallucinated)
	}
}

func TestValidateCitations_ReplacesHallucinated(t *testing.T) {
	allowed := map[string]bool{"123": true}
	in := "Per (PMID: 123) and the well-known (PMID: 999) trial."

	out, hallucinated := validateCitations(in, allowed)

	if !strings.Contains(out, "PMID: 123") {
		t.Errorf("real PMID should remain, got %q", out)
	}
	if strings.Contains(out, "PMID: 999") {
		t.Errorf("fake PMID should be replaced, got %q", out)
	}
	if !strings.Contains(out, "[unverified]") {
		t.Errorf("expected [unverified] marker, got %q", out)
	}
	if len(hallucinated) != 1 || hallucinated[0] != "999" {
		t.Errorf("expected [999], got %v", hallucinated)
	}
}

func TestValidateCitations_NoCitations(t *testing.T) {
	out, hallucinated := validateCitations("No citations here.", map[string]bool{})
	if out != "No citations here." {
		t.Errorf("text should be unchanged, got %q", out)
	}
	if len(hallucinated) != 0 {
		t.Errorf("expected no hallucinations, got %v", hallucinated)
	}
}

func TestAppendAPAReferences_OnlyCitedPapersIncluded(t *testing.T) {
	papers := []pubmed.Paper{
		{PMID: "123", Title: "Trial A", Year: "2024", Journal: "NEJM", Authors: []string{"Smith J", "Doe A"}},
		{PMID: "456", Title: "Trial B", Year: "2023", Journal: "Lancet", Authors: []string{"Lee K"}},
	}
	text := "Smith et al. (PMID: 123) showed efficacy."

	out := appendAPAReferences(text, papers)

	if !strings.Contains(out, "References (APA):") {
		t.Errorf("expected References section, got %q", out)
	}
	if !strings.Contains(out, "Trial A") {
		t.Errorf("cited paper Trial A should appear, got %q", out)
	}
	if strings.Contains(out, "Trial B") {
		t.Errorf("uncited paper Trial B should not appear, got %q", out)
	}
}

func TestAppendAPAReferences_NoCitationsReturnsOriginal(t *testing.T) {
	papers := []pubmed.Paper{
		{PMID: "123", Title: "Trial A", Year: "2024", Journal: "NEJM"},
	}
	text := "No PMIDs cited in this text."

	out := appendAPAReferences(text, papers)

	if out != text {
		t.Errorf("expected unchanged text, got %q", out)
	}
}

func TestAppendAPAReferences_NoPapersReturnsOriginal(t *testing.T) {
	text := "Per (PMID: 123)."
	out := appendAPAReferences(text, nil)
	if out != text {
		t.Errorf("expected unchanged text, got %q", out)
	}
}

func TestAppendAPAReferences_EtAlForManyAuthors(t *testing.T) {
	papers := []pubmed.Paper{
		{PMID: "123", Title: "T", Year: "2024", Journal: "J",
			Authors: []string{"A", "B", "C", "D", "E"}},
	}
	out := appendAPAReferences("(PMID: 123)", papers)
	if !strings.Contains(out, "et al.") {
		t.Errorf("expected et al. for 5 authors, got %q", out)
	}
}
