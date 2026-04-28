package handlers

import (
	"cancer-to-hell/agents"
	"cancer-to-hell/pubmed"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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

	// Step 4 — fetch real papers from PubMed first
	searchQuery := input.CancerType + " " +
		strings.Join(input.Biomarkers, " ") +
		" treatment"

	papers, err := pubmed.Search(searchQuery)
	if err != nil {
		papers = []pubmed.Paper{}
	}

	// Step 5 — append real papers to patient context
	// This forces agents to cite from real sources, not invented ones
	if len(papers) > 0 {
		patientContext += "\nREAL PEER-REVIEWED PAPERS FOR CITATION:\n"
		for _, p := range papers {
			patientContext += fmt.Sprintf(
				"- %s. %s (%s). https://pubmed.ncbi.nlm.nih.gov/%s\n",
				p.Title, p.Journal, p.Year, p.PMID,
			)
		}
		patientContext += "\nYou MUST cite only from the papers listed above. Do not invent citations.\n"
	}

	// Step 6 — call all 3 clinical modules
	evidence, err := agents.EvidenceRetrieval(patientContext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Evidence module failed: " + err.Error()})
		return
	}

	guideline, err := agents.GuidelineAlignment(patientContext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Guideline module failed: " + err.Error()})
		return
	}

	safety, err := agents.SafetyRisk(patientContext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Safety module failed: " + err.Error()})
		return
	}

	// Step 7 — return everything together
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"evidence":  evidence,
		"guideline": guideline,
		"safety":    safety,
		"papers":    papers,
	})
}
