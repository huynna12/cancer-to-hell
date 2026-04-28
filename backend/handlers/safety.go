package handlers

import "strings"

// PatientInput is the full patient profile sent from the frontend
type PatientInput struct {
	CancerType    string   `json:"cancer_type" binding:"required"`
	Stage         string   `json:"stage" binding:"required"`
	Biomarkers    []string `json:"biomarkers"`
	ECOG          string   `json:"ecog"`
	PriorLines    []string `json:"prior_lines"`
	Comorbidities []string `json:"comorbidities"`
	CurrentMeds   []string `json:"current_meds"`
	Labs          Labs     `json:"labs"`
}

// Labs contains critical organ function data
type Labs struct {
	EGFR       string `json:"egfr"`
	LiverPanel string `json:"liver_panel"`
	CBC        string `json:"cbc"`
	ECG        string `json:"ecg"`
}

// CriticalMissingData checks for required fields before allowing a recommendation
// If anything critical is missing — we block, not guess
func CriticalMissingData(input PatientInput) []string {
	missing := make([]string, 0)

	if strings.TrimSpace(input.ECOG) == "" {
		missing = append(missing, "ECOG performance status")
	}
	if strings.TrimSpace(input.Labs.EGFR) == "" {
		missing = append(missing, "eGFR (kidney function)")
	}
	if strings.TrimSpace(input.Labs.LiverPanel) == "" {
		missing = append(missing, "Liver panel")
	}
	if strings.TrimSpace(input.Labs.CBC) == "" {
		missing = append(missing, "CBC (complete blood count)")
	}

	return missing
}

// BuildPatientContext converts the patient input into a string
// that all 3 clinical modules receive as their problem to analyze
func BuildPatientContext(input PatientInput) string {
	context := "PATIENT PROFILE:\n"
	context += "Cancer Type: " + input.CancerType + "\n"
	context += "Stage: " + input.Stage + "\n"
	context += "Biomarkers: " + strings.Join(input.Biomarkers, ", ") + "\n"
	context += "ECOG: " + input.ECOG + "\n"
	context += "Prior Lines: " + strings.Join(input.PriorLines, ", ") + "\n"
	context += "Labs — eGFR: " + input.Labs.EGFR + "\n"
	context += "Labs — Liver Panel: " + input.Labs.LiverPanel + "\n"
	context += "Labs — CBC: " + input.Labs.CBC + "\n"
	context += "Labs — ECG: " + input.Labs.ECG + "\n"

	if len(input.Comorbidities) > 0 {
		context += "Comorbidities: " + strings.Join(input.Comorbidities, ", ") + "\n"
	}
	if len(input.CurrentMeds) > 0 {
		context += "Current Medications: " + strings.Join(input.CurrentMeds, ", ") + "\n"
	}

	return context
}
