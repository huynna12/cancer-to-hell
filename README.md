# Cancer to Hell

Multi-agent clinical decision support prototype for breast cancer tumor-board reasoning. Three specialist LLM agents run concurrently against a patient case, ground their reasoning in live PubMed abstracts, and stream partial results to the UI over Server-Sent Events. Hallucinated citations are caught and flagged before reaching the user. Pre-trained clinical knowledge is explicitly attributed with `(Gemini)` so readers can distinguish retrieved evidence from model knowledge.

> **Research / educational use only.** This is a portfolio project demonstrating LLM orchestration patterns. It is **not** a medical device, has not been clinically validated, and must not be used to make treatment decisions. Submit synthetic patient profiles only — do not enter real PHI.

## Live demo

- **Frontend:** https://cancer-to-hell.vercel.app
- **Backend health:** https://cancer-to-hell-api.fly.dev/health

Specialized for **breast cancer** decision support across all stages (early, locally advanced, metastatic). The orchestration pattern generalizes to other solid tumors but agent prompts are tuned for breast cancer.

## Highlights

- **Concurrent multi-agent orchestration** — Evidence Retrieval, Guideline Alignment, and Safety/Risk specialists run as Go goroutines coordinated via `sync.WaitGroup` + buffered channels.
- **Live PubMed RAG** — fresh abstracts retrieved per case from the NCBI E-utilities API, biomarker-aware query construction with cleaned terms and fallback on zero results.
- **Hallucination guardrail** — every `PMID` in model output is validated against the retrieved set via regex; fabricated PMIDs are replaced with `[unverified]` and surfaced to the user.
- **Provenance-aware attribution** — claims from retrieved papers cite as `(Author et al., Year. PMID: 28578601)`; claims from the model's pre-trained knowledge are tagged `(Gemini)`. Author-year citations without PMIDs are explicitly forbidden so readers can always verify.
- **Safety pre-check** — required clinical inputs (ECOG, eGFR, liver panel, CBC) are validated *before* any LLM call. Missing data → blocked response, no tokens burned.
- **SSE streaming UI** — partial module outputs render as soon as each agent finishes; users don't wait for the slowest agent.
- **Quantitative evaluation** — n=8 case eval against NCCN Breast Cancer Guidelines with a 4-axis rubric (regimen / biomarker reasoning / safety / citations). See [`eval/RESULTS.md`](eval/RESULTS.md).

## Architecture

```
                  ┌────────────────────┐
                  │  Next.js frontend  │
                  │  (Vercel)          │
                  └─────────┬──────────┘
                            │ POST /api/v1/decision-card/stream  (SSE)
                            ▼
                  ┌────────────────────┐
                  │   Go / Gin API     │
                  │   (Fly.io)         │
                  └─────────┬──────────┘
                            │
                  Safety pre-check (ECOG, labs)
                            │
                  PubMed retrieval + caching
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
  Evidence agent      Guideline agent      Safety agent
        │                   │                   │
        └─────────┬─────────┴─────────┬─────────┘
                  ▼                   ▼
            Citation validation     Stream module events (SSE)
            (PMID guardrail)
                  │
                  ▼
            Final response
```

Each agent receives the same patient context (profile + retrieved abstracts) and produces a 3–5 sentence paragraph in its specialty. Agents run sequentially in the SSE handler to stay under the Google AI Studio rate limit; the streaming response keeps perceived latency low.

## Tech stack

| Layer    | Tech |
|----------|------|
| Frontend | Next.js 16, React 19, Tailwind v4, react-markdown |
| Backend  | Go 1.26, Gin, Server-Sent Events |
| LLM      | Google Gemini 2.5 Flash Lite (via Google AI Studio) |
| Data     | NCBI PubMed E-utilities (esearch + efetch) |
| Infra    | Fly.io (Go API, Docker), Vercel (frontend) |

## Evaluation

The model was evaluated on n=8 synthetic breast cancer cases spanning early-stage (adjuvant decisions), locally advanced (neoadjuvant), and metastatic (multiple lines of therapy across HER2+, TNBC, and HR+ subtypes). Reference treatments were extracted from NCCN Breast Cancer Guidelines *before* model runs to prevent post-hoc bias.

Each case was scored on four axes (0–2 each, total /8):

- **Regimen choice** — does the recommended regimen match the NCCN-preferred or category-1 option?
- **Biomarker reasoning** — does the model correctly use the biomarker profile to drive its choice?
- **Safety adjustments** — does it flag dose modifications, contraindications, monitoring needs?
- **Citation validity** — are PMIDs real and on-topic? Are non-paper claims tagged `(Gemini)`?

Full per-case scoring, raw model outputs, and methodology are in [`eval/RESULTS.md`](eval/RESULTS.md).

## Run locally

### Backend

```bash
cd backend
cat > .env <<'EOF'
GOOGLE_API_KEY=your-google-ai-studio-key   # required
NCBI_API_KEY=optional-ncbi-key             # optional, raises PubMed rate limit
GEMMA_MODEL=gemini-2.5-flash-lite          # any Google AI Studio model name
EOF
go run .
```

The API listens on `:8080` by default. Get a Google API key from https://aistudio.google.com/apikey.

### Frontend

```bash
cd frontend
npm install
NEXT_PUBLIC_BACKEND_URL=http://localhost:8080 npm run dev
```

Open `http://localhost:3000`.

### Tests

```bash
cd backend
go test ./...
```

## Deploy

### Backend → Fly.io

```bash
cd backend
fly launch --no-deploy --copy-config              # picks up fly.toml
fly secrets set GOOGLE_API_KEY=...                # required
fly secrets set NCBI_API_KEY=...                  # optional
fly secrets set GEMMA_MODEL=gemini-2.5-flash-lite # or any Google AI Studio model
fly secrets set FRONTEND_ORIGIN_SUFFIX=.vercel.app # allows any *.vercel.app origin
fly deploy
```

### Frontend → Vercel

1. Import the GitHub repo into Vercel; set **Root Directory** to `frontend/`.
2. Set **Framework Preset** to Next.js (auto-detection sometimes misses Next 16).
3. Set environment variable `NEXT_PUBLIC_BACKEND_URL` to your Fly app URL (e.g. `https://cancer-to-hell-api.fly.dev`).
4. Deploy.

CORS uses `FRONTEND_ORIGIN_SUFFIX=.vercel.app` so every Vercel preview URL works automatically without per-deploy reconfiguration.

## API

### `POST /api/v1/decision-card/stream`

SSE: emits `status`, `module`, and `done` events as each agent finishes. Each `done` event includes any hallucinated PMIDs that were stripped from the output.

```bash
curl -N -X POST $BACKEND_URL/api/v1/decision-card/stream \
  -H "Content-Type: application/json" \
  -d '{
    "cancer_type": "Breast Cancer",
    "stage": "Stage IV (metastatic)",
    "biomarkers": ["HER2-positive", "ER-positive"],
    "ecog": "1",
    "prior_lines": ["Adjuvant AC-T + trastuzumab"],
    "comorbidities": ["Hypertension"],
    "current_meds": ["Lisinopril"],
    "labs": {
      "egfr": "85",
      "liver_panel": "AST 28, ALT 32, bili 0.8",
      "cbc": "Hgb 12.4, WBC 6.2, plt 245k",
      "ecg": "NSR, LVEF 58%"
    }
  }'
```

### `POST /api/v1/decision-card`

Synchronous version: runs all three agents and returns one JSON payload.

### `GET /health`

Liveness probe. Returns `200 {"status": "Cancer to Hell is running"}`.

## Project layout

```
backend/
  agents/         # specialist prompts and LLM calls
  gemma/          # Google AI Studio HTTP client (with retry)
  handlers/       # HTTP handlers, citation validation, SSE, query construction
  pubmed/         # NCBI E-utilities client + structured-abstract parser
  main.go
  Dockerfile
  fly.toml
frontend/
  app/            # Next.js app router
eval/
  RESULTS.md      # per-case scoring against NCCN reference treatments
```

## Limitations and future work

- **Synthetic cases only** — n=8 author-constructed scenarios, not real patients. Real-world complexity (patient preference, trial availability, comorbidity interactions) is not modeled.
- **Single cancer type** — agent prompts are tuned for breast cancer. Other malignancies would need re-tuned prompts and stage-specific decision logic.
- **PubMed retrieval has gaps** — query construction uses cancer type + cleaned biomarkers + "treatment". Some landmark trials don't surface, in which case the model falls back to `(Gemini)`-attributed pre-trained knowledge.
- **No clinical validation** — guideline-concordance is not safety. This tool would need IRB-approved prospective testing before any clinical use.
- **Future:** structured outputs (JSON schema), agent-to-agent disagreement resolution, larger eval set with held-out validation, broader cancer coverage.

## License

MIT
