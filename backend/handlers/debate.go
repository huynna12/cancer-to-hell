package handlers

import (
	"cancer-to-hell/agents"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DebateRequest struct {
	CancerType string `json:"cancer_type" binding:"required"`
	Stage      string `json:"stage" binding:"required"`
	Mutations  string `json:"mutations" binding:"required"`
}

type DebateResponse struct {
	Evidence  string `json:"evidence"`
	Guideline string `json:"guideline"`
	Safety    string `json:"safety"`
}

func StartDebate(c *gin.Context) {
	var req DebateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	patientContext := "Cancer Type: " + req.CancerType +
		"\nStage: " + req.Stage +
		"\nKnown Mutations: " + req.Mutations

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

	c.JSON(http.StatusOK, DebateResponse{
		Evidence:  evidence,
		Guideline: guideline,
		Safety:    safety,
	})
}
