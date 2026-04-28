package handlers

import (
	"cancer-to-hell/agents"
	"net/http"

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
	// If critical data is missing, block immediately without calling Gemma 4
	missing := CriticalMissingData(input)
	if len(missing) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":       "blocked",
			"missing_data": missing,
			"message":      "Safety pre-check blocked recommendations. Please provide the missing data before proceeding.",
		})
		return
	}

	// Step 3 — build patient context string for all 3 modules
	patientContext := BuildPatientContext(input)

	// Step 4 — call all 3 clinical modules sequentially for now
	// (concurrent goroutines coming in next commit)
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

	// Step 5 — return all 3 responses
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"evidence":  evidence,
		"guideline": guideline,
		"safety":    safety,
	})
}
