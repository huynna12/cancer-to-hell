package agents

import (
	"cancer-to-hell/gemma"
	"regexp"
	"strings"
)

// thinkingLine strips lines that start with * — the model's leaked scratchpad bullets.
var thinkingLine = regexp.MustCompile(`(?m)^\s*\*[^\n]*\n?`)
var metaLine = regexp.MustCompile(`(?mi)^\s*(sentence\s*\d+|total\s*:|single paragraph\?|no headers|pmids?\s*correct|word count|final check|self-correction|final polish|drafting citations|wait,|perfect\.?|correct\.?|citation[s]? check)\b[^\n]*\n?`)
var numberedLine = regexp.MustCompile(`(?m)^\s*\d+\.\s+`)
var headingLine = regexp.MustCompile(`(?mi)^\s*(evidence retrieval|guideline alignment|references?\s*\(apa\)?)\s*$`)
var sentenceBoundary = regexp.MustCompile(`(?m)([.!?])\s+`)
var inlineSpace = regexp.MustCompile(`\s+`)

func uniqueSentences(sentences []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(sentences))
	for _, s := range sentences {
		norm := strings.ToLower(strings.TrimSpace(s))
		if norm == "" || seen[norm] {
			continue
		}
		seen[norm] = true
		result = append(result, strings.TrimSpace(s))
	}
	return result
}

func cleanResponse(response string) string {
	response = thinkingLine.ReplaceAllString(response, "")
	response = metaLine.ReplaceAllString(response, "")
	response = headingLine.ReplaceAllString(response, "")
	response = numberedLine.ReplaceAllString(response, "")
	response = inlineSpace.ReplaceAllString(response, " ")
	response = strings.ReplaceAll(response, " .", ".")
	response = strings.ReplaceAll(response, " ,", ",")
	response = strings.TrimSpace(response)

	parts := strings.FieldsFunc(response, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	if len(parts) > 0 {
		response = strings.TrimSpace(parts[0])
	}

	chunks := sentenceBoundary.Split(response, -1)
	chunks = uniqueSentences(chunks)
	if len(chunks) > 5 {
		chunks = chunks[:5]
	}
	response = strings.Join(chunks, ". ")
	if response != "" && !strings.HasSuffix(response, ".") {
		response += "."
	}
	blank := regexp.MustCompile(`\n{3,}`)
	response = blank.ReplaceAllString(response, "\n\n")
	return strings.TrimSpace(response)
}

func EvidenceRetrieval(patientContext string) (string, error) {
	system := `You are an oncology research scientist specializing in metastatic breast cancer.

RULE: Output ONLY a single flowing paragraph. No headers, no bullet points, no numbered lists, no self-check notes, and no chain-of-thought. Start directly with the clinical reasoning.

Write 4-6 sentences that reason through the strongest evidence for treating THIS specific patient. Weave citations inline where they exist. For every study you cite, embed the PMID at the end of the sentence like this: (Author et al., Year. PMID: 28578601)

Your reasoning should:
- Connect the patient's specific biomarkers and prior therapy to the evidence
- End with a brief note on evidence quality (RCT, retrospective, etc.)

CITATION RULES — read carefully:
- Cite ONLY from papers listed under "REAL PEER-REVIEWED PAPERS" in the patient context
- Use the exact PMID numbers provided — do not change or guess them
- If no papers are listed in the context, write the reasoning without any citations
- Never invent PMIDs, author names, trial names, or journals`

	result, err := gemma.Ask(system, patientContext)
	return cleanResponse(result), err
}

func GuidelineAlignment(patientContext string) (string, error) {
	system := `You are a clinical oncologist specializing in metastatic breast cancer guidelines.

RULE: Output ONLY a single flowing paragraph. No headers, no bullet points, no numbered lists, no self-check notes, and no chain-of-thought. Start directly with the clinical reasoning.

Write 4-6 sentences that map THIS specific patient to guideline-concordant treatment pathways. Your reasoning should:
- Name the line of therapy (first-line, second-line, etc.) and why
- List the top 2-3 regimen options woven naturally into the prose, with a brief rationale for each
- Reference the relevant guideline body (NCCN, ASCO, ESMO) and year
- Embed any relevant PMID inline like: (Author et al., Year. PMID: 28578601)
- End with a clear recommendation for this specific patient given their biomarkers and prior therapy

CITATION RULES — read carefully:
- Cite ONLY from papers listed under "REAL PEER-REVIEWED PAPERS" in the patient context
- Use the exact PMID numbers provided — do not change or guess them
- If no papers are listed in the context, write the reasoning without any citations
- Never invent PMIDs, author names, trial names, or journals`

	result, err := gemma.Ask(system, patientContext)
	return cleanResponse(result), err
}
