package agents

import (
	"cancer-to-hell/gemma"
	"regexp"
	"strings"
)

// thinkingLine strips lines that start with * — the model's leaked scratchpad bullets.
var thinkingLine = regexp.MustCompile(`(?m)^\s*\*[^\n]*\n?`)

func cleanResponse(response string) string {
	response = thinkingLine.ReplaceAllString(response, "")
	blank := regexp.MustCompile(`\n{3,}`)
	response = blank.ReplaceAllString(response, "\n\n")
	return strings.TrimSpace(response)
}

func EvidenceRetrieval(patientContext string) (string, error) {
	system := `You are an oncology research scientist specializing in metastatic breast cancer.

RULE: Output ONLY a single flowing paragraph. No headers, no bullet points, no numbered lists. No preamble like "Based on..." or "I will...". Start directly with the clinical reasoning.

Write 4-6 sentences that reason through the strongest evidence for treating THIS specific patient. Weave citations inline where they exist. For every study you cite, embed the PMID at the end of the sentence like this: (Author et al., Year. PMID: 28578601)

Your reasoning should:
- Connect the patient's specific biomarkers and prior therapy to the evidence
- Name the most relevant RCT or trial by its common name (e.g. OlympiAD, MONARCH-2)
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

RULE: Output ONLY a single flowing paragraph. No headers, no bullet points, no numbered lists. No preamble like "Based on..." or "I will...". Start directly with the clinical reasoning.

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
