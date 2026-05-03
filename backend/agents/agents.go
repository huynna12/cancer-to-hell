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

// malformedCitation matches sentences using the bad "per Author et al." form
// instead of the proper trailing (Author et al., Year. PMID: XXXXXXX) format.
var malformedCitation = regexp.MustCompile(`(?i)\bper\s+[A-Z][a-zA-Z\-']+\s+et\s+al`)

func filterMalformed(sentences []string) []string {
	out := make([]string, 0, len(sentences))
	for _, s := range sentences {
		if malformedCitation.MatchString(s) {
			continue
		}
		out = append(out, s)
	}
	return out
}

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
	chunks = filterMalformed(chunks)
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
	system := `You are an oncology research scientist specializing in breast cancer across all stages (early, locally advanced, and metastatic).

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

func SafetyRisk(patientContext string) (string, error) {
	system := `You are a clinical pharmacologist specializing in oncology risk assessment.

RULE: Output ONLY a single flowing paragraph of 3-4 sentences. No headers, no bullet points, no numbered lists, no regimen summary list, no self-check notes, and no chain-of-thought. Start directly with the risk reasoning.

For the 2-3 regimens most likely to be considered for THIS specific patient, state the single most important risk or adverse event for each, in prose. Account for this patient's labs, comorbidities, and current medications when relevant.

CITATION FORMAT — strict:
- Citations go at the END of a complete sentence in this exact format: (Author et al., Year. PMID: 28578601)
- NEVER write a citation inside another parenthetical
- NEVER use the word "per" before an author name
- NEVER nest parentheses

CONTENT RULES:
- Cite ONLY from papers listed under "REAL PEER-REVIEWED PAPERS" in the patient context
- If a risk statement comes from general clinical knowledge rather than a listed paper, write the reasoning without a citation
- Never invent PMIDs, author names, trial names, or journals`

	result, err := gemma.Ask(system, patientContext)
	return cleanResponse(result), err
}

func GuidelineAlignment(patientContext string) (string, error) {
	system := `You are a clinical oncologist specializing in breast cancer treatment guidelines across all stages (early, locally advanced, and metastatic).

RULE: Output ONLY a single flowing paragraph. No headers, no bullet points, no numbered lists, no regimen summary list, no self-check notes, and no chain-of-thought. Start directly with the clinical reasoning.

Write 4-6 sentences that map THIS specific patient to guideline-concordant treatment pathways. Your reasoning should:
- Name the line of therapy (first-line, second-line, etc.) and why
- Discuss the top 2-3 regimen options as part of the prose, each with a brief rationale
- Reference the relevant guideline body (NCCN, ASCO, ESMO) and year
- End with a clear recommendation for this specific patient given their biomarkers and prior therapy

CITATION FORMAT — strict:
- Citations go at the END of a complete sentence in this exact format: (Author et al., Year. PMID: 28578601)
- NEVER write a citation inside another parenthetical, like "(rationale per Author et al., Year, PMID: ...)"
- NEVER use the word "per" before an author name — use a complete sentence and put the citation at the end
- NEVER nest parentheses

CONTENT RULES:
- Cite ONLY from papers listed under "REAL PEER-REVIEWED PAPERS" in the patient context
- Use the exact PMID numbers provided — do not change or guess them
- If no papers are listed in the context, write the reasoning without any citations
- Never invent PMIDs, author names, trial names, or journals`

	result, err := gemma.Ask(system, patientContext)
	return cleanResponse(result), err
}
