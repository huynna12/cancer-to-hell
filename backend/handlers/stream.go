package handlers

import (
	"cancer-to-hell/agents"
	"cancer-to-hell/pubmed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type sseEvent struct {
	Type                  string   `json:"type"`
	Name                  string   `json:"name,omitempty"`
	Content               string   `json:"content,omitempty"`
	Message               string   `json:"message,omitempty"`
	MissingData           []string `json:"missing_data,omitempty"`
	HallucinatedCitations []string `json:"hallucinated_citations,omitempty"`
}

func StreamDebate(c *gin.Context) {
	var input PatientInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	missing := CriticalMissingData(input)
	if len(missing) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":       "blocked",
			"missing_data": missing,
			"message":      "Safety pre-check blocked recommendations. Please provide the missing data before proceeding.",
		})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	eventCh := make(chan sseEvent, 10)

	go streamDebateEvents(input, eventCh)

	c.Stream(func(w io.Writer) bool {
		event, ok := <-eventCh
		if !ok {
			return false
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		return true
	})
}

func streamDebateEvents(input PatientInput, eventCh chan<- sseEvent) {
	defer close(eventCh)

	eventCh <- sseEvent{Type: "status", Message: "Fetching PubMed papers..."}

	papers, allowedPMIDs := fetchPapers(input)
	patientContext := buildContext(input, papers)
	eventCh <- sseEvent{Type: "status", Message: fmt.Sprintf("Running clinical modules (found %d papers)...", len(papers))}

	_, allHallucinated := runClinicalModules(patientContext, allowedPMIDs, papers, eventCh)

	done := sseEvent{Type: "done"}
	if len(allHallucinated) > 0 {
		done.HallucinatedCitations = allHallucinated
	}
	eventCh <- done
}

func fetchPapers(input PatientInput) ([]pubmed.Paper, map[string]bool) {
	searchQuery := input.CancerType + " " + strings.Join(input.Biomarkers, " ") + " treatment"
	papers, err := pubmed.Search(searchQuery)
	if err != nil {
		papers = []pubmed.Paper{}
	}

	allowedPMIDs := make(map[string]bool, len(papers))
	for _, p := range papers {
		allowedPMIDs[p.PMID] = true
	}

	return papers, allowedPMIDs
}

func runClinicalModules(patientContext string, allowedPMIDs map[string]bool, papers []pubmed.Paper, eventCh chan<- sseEvent) (map[string]string, []string) {
	modules := []struct {
		name string
		fn   func(string) (string, error)
	}{
		{"evidence", agents.EvidenceRetrieval},
		{"guideline", agents.GuidelineAlignment},
		{"safety", agents.SafetyRisk},
	}

	outputs := make(map[string]string)
	var allHallucinated []string

	// Sequential: each module gets a clean turn with no concurrent API pressure.
	// Running all three in parallel saturates the free-tier rate limit and causes
	// the second and third requests to queue past the 5-minute timeout.
	for _, mod := range modules {
		result, err := mod.fn(patientContext)
		content := result
		if err != nil {
			content = "Module failed: " + err.Error()
		}

		cleaned, found := validateCitations(content, allowedPMIDs)
		withRefs := appendAPAReferences(cleaned, papers)
		outputs[mod.name] = withRefs
		allHallucinated = append(allHallucinated, found...)

		evt := sseEvent{Type: "module", Name: mod.name, Content: withRefs}
		if len(found) > 0 {
			evt.HallucinatedCitations = found
		}
		eventCh <- evt
	}

	return outputs, allHallucinated
}
