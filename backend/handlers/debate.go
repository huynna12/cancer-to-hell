package handlers

import (
	"cancer-to-hell/agents"
	"cancer-to-hell/pubmed"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type moduleOutput struct {
	name   string
	result string
	err    error
}

var pmidPattern = regexp.MustCompile(`PMID:\s*(\d+)`)

func appendAPAReferences(text string, papers []pubmed.Paper) string {
	if len(papers) == 0 {
		return text
	}

	cited := map[string]bool{}
	for _, m := range pmidPattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			cited[m[1]] = true
		}
	}
	if len(cited) == 0 {
		return text
	}

	refs := make([]string, 0, len(cited))
	for _, p := range papers {
		if !cited[p.PMID] {
			continue
		}
		authorStr := "Unknown Authors"
		if len(p.Authors) > 3 {
			authorStr = strings.Join(p.Authors[:3], ", ") + ", et al."
		} else if len(p.Authors) > 0 {
			authorStr = strings.Join(p.Authors, ", ")
		}
		refs = append(refs, fmt.Sprintf("- %s (%s). %s. %s. https://pubmed.ncbi.nlm.nih.gov/%s/", authorStr, p.Year, p.Title, p.Journal, p.PMID))
	}
	if len(refs) == 0 {
		return text
	}

	return strings.TrimSpace(text) + "\n\nReferences (APA):\n" + strings.Join(refs, "\n")
}

// validateCitations replaces PMIDs not in the fetched paper set with [unverified].
func validateCitations(text string, allowedPMIDs map[string]bool) (string, []string) {
	var hallucinated []string
	result := pmidPattern.ReplaceAllStringFunc(text, func(match string) string {
		sub := pmidPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		pmid := sub[1]
		if allowedPMIDs[pmid] {
			return match
		}
		hallucinated = append(hallucinated, pmid)
		return "PMID: [unverified]"
	})
	return result, hallucinated
}

func buildContext(input PatientInput, papers []pubmed.Paper) string {
	patientContext := BuildPatientContext(input)

	if len(papers) == 0 {
		return patientContext
	}

	patientContext += "\nREAL PEER-REVIEWED PAPERS — READ THE ABSTRACTS TO GROUND YOUR REASONING:\n"
	for i, p := range papers {
		authorStr := "Unknown Authors"
		if len(p.Authors) > 3 {
			authorStr = strings.Join(p.Authors[:3], ", ") + ", et al."
		} else if len(p.Authors) > 0 {
			authorStr = strings.Join(p.Authors, ", ")
		}
		patientContext += fmt.Sprintf(
			"\n[%d] %s (%s). %s. %s. PMID: %s\n",
			i+1, authorStr, p.Year, p.Title, p.Journal, p.PMID,
		)
		if p.Abstract != "" {
			patientContext += fmt.Sprintf("    Abstract: %s\n", p.Abstract)
		}
	}
	patientContext += "\nINSTRUCTIONS FOR USING THESE PAPERS:\n" +
		"- These are the most recent peer-reviewed studies retrieved specifically for this patient case.\n" +
		"- Base your clinical reasoning on what the abstracts actually found — read the RESULTS and CONCLUSIONS sections.\n" +
		"- If a retrieved abstract contradicts your training knowledge or older guidelines, DEFER TO THE ABSTRACT. Recent evidence overrides older data.\n" +
		"- Cite using the exact author names, year, and PMID provided above.\n" +
		"- Do not invent citations or use PMIDs not listed here.\n"

	return patientContext
}

func runModules(patientContext string) (map[string]string, error) {
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

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	outputs := make(map[string]string)
	for output := range resultCh {
		if output.err != nil {
			outputs[output.name] = "Module failed: " + output.err.Error()
		} else {
			outputs[output.name] = output.result
		}
	}
	return outputs, nil
}

func StartDebate(c *gin.Context) {
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

	searchQuery := buildSearchQuery(input.CancerType, input.Biomarkers)
	papers, err := pubmed.Search(searchQuery)
	if err != nil || len(papers) == 0 {
		fallback := input.CancerType + " treatment"
		if len(input.Biomarkers) > 0 {
			fallback = input.CancerType + " " + cleanTerm(input.Biomarkers[0]) + " treatment"
		}
		papers, err = pubmed.Search(fallback)
		if err != nil {
			papers = []pubmed.Paper{}
		}
	}

	allowedPMIDs := make(map[string]bool, len(papers))
	for _, p := range papers {
		allowedPMIDs[p.PMID] = true
	}

	patientContext := buildContext(input, papers)
	outputs, _ := runModules(patientContext)

	var allHallucinated []string
	for name, text := range outputs {
		cleaned, found := validateCitations(text, allowedPMIDs)
		outputs[name] = appendAPAReferences(cleaned, papers)
		allHallucinated = append(allHallucinated, found...)
	}

	resp := gin.H{
		"status":    "ok",
		"evidence":  outputs["evidence"],
		"guideline": outputs["guideline"],
		"safety":    outputs["safety"],
	}
	if len(allHallucinated) > 0 {
		resp["hallucinated_citations"] = allHallucinated
	}

	c.JSON(http.StatusOK, resp)
}
