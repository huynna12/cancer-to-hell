package handlers

import (
	"cancer-to-hell/agents"
	"cancer-to-hell/pubmed"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// moduleOutput holds the result from a single clinical module
type moduleOutput struct {
	name   string
	result string
	err    error
}

func StartDebate(c *gin.Context) {
	// Step 1 — parse the incoming request
	var input PatientInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Step 2 — safety pre-check
	missing := CriticalMissingData(input)
	if len(missing) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":       "blocked",
			"missing_data": missing,
			"message":      "Safety pre-check blocked recommendations. Please provide the missing data before proceeding.",
		})
		return
	}

	// Step 3 — build patient context
	patientContext := BuildPatientContext(input)

	// Step 4 — fetch real papers from PubMed
	searchQuery := input.CancerType + " " +
		strings.Join(input.Biomarkers, " ") +
		" treatment"

	papers, err := pubmed.Search(searchQuery)
	if err != nil {
		papers = []pubmed.Paper{}
	}

	// Step 5 — append real papers to patient context
	if len(papers) > 0 {
		patientContext += "\nREAL PEER-REVIEWED PAPERS FOR CITATION:\n"
		for _, p := range papers {
			authorStr := "Unknown Authors"
			if len(p.Authors) > 3 {
				authorStr = strings.Join(p.Authors[:3], ", ") + ", et al."
			} else if len(p.Authors) > 0 {
				authorStr = strings.Join(p.Authors, ", ")
			}
			patientContext += fmt.Sprintf(
				"- %s (%s). %s. %s. PMID: %s\n",
				authorStr, p.Year, p.Title, p.Journal, p.PMID,
			)
		}
		patientContext += "\nCite in APA format using the author names, year, title, journal and PMID provided above. Do not invent citations.\n"
	}

	// Step 6 — run all 3 modules concurrently using goroutines
	// Each module runs in its own goroutine simultaneously
	var wg sync.WaitGroup
	resultCh := make(chan moduleOutput, 3)

	modules := []struct {
		name string
		fn   func(string) (string, error)
	}{
		{"evidence", agents.EvidenceRetrieval},
		{"guideline", agents.GuidelineAlignment},
		{"safety", agents.SafetyRisk},
	}

	for _, mod := range modules {
		wg.Add(1)
		go func(name string, fn func(string) (string, error)) {
			defer wg.Done()
			result, err := fn(patientContext)
			resultCh <- moduleOutput{name: name, result: result, err: err}
		}(mod.name, mod.fn)
	}

	// Wait for all goroutines to finish then close the channel
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Step 7 — collect results from channel
	outputs := make(map[string]string)
	for output := range resultCh {
		if output.err != nil {
			outputs[output.name] = "Module failed: " + output.err.Error()
		} else {
			outputs[output.name] = output.result
		}
	}

	// Step 8 — return everything
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"evidence":  outputs["evidence"],
		"guideline": outputs["guideline"],
		"safety":    outputs["safety"],
		"papers":    papers,
	})
}
