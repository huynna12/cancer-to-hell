# Cancer to Hell

Multi-agent clinical decision support prototype for metastatic breast cancer tumor-board reasoning. Three specialist LLM agents run concurrently against a patient case, ground their reasoning in live PubMed abstracts, and stream partial results to the UI over Server-Sent Events. Hallucinated citations are caught and flagged before reaching the user.

> **Research / educational use only.** This is a portfolio project demonstrating LLM orchestration patterns. It is **not** a medical device, has not been clinically validated, and must not be used to make treatment decisions.

## Highlights

- **Concurrent multi-agent orchestration** — Evidence Retrieval, Guideline Alignment, and Safety/Risk specialists run in parallel via `sync.WaitGroup` + buffered channels in Go.
- **Live PubMed RAG** — fresh abstracts are retrieved per case from the NCBI E-utilities API and injected into each agent's context.
- **Citation grounding with hallucination guardrail** — every `PMID` in model output is validated against the retrieved set; fabricated PMIDs are replaced with `[unverified]` and surfaced to the user.
- **SSE streaming UI** — partial module outputs appear as soon as each agent finishes, instead of waiting for the slowest one.
- **Safety pre-check** — required clinical inputs (ECOG, eGFR, etc.) are validated before any LLM call is made.

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
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
  Evidence agent      Guideline agent      Safety agent
        │                   │                   │
        └─────────┬─────────┴─────────┬─────────┘
                  ▼                   ▼
            Gemini API           PubMed E-utilities
                                 (cached per query)
```

## Tech stack

| Layer    | Tech |
|----------|------|
| Frontend | Next.js 16, React 19, Tailwind v4, react-markdown |
| Backend  | Go 1.26, Gin, Server-Sent Events |
| LLM      | Google Gemini (via `generativelanguage.googleapis.com`) |
| Data     | NCBI PubMed E-utilities (esearch + efetch) |
| Infra    | Fly.io (Go API, Docker), Vercel (frontend) |

## Run locally

### Backend

```bash
cd backend
echo 'GEMINI_API_KEY=your-key-here' > .env   # required
echo 'NCBI_API_KEY=optional-key' >> .env     # optional, raises PubMed rate limit
go run .
```

The API listens on `:8080` by default.

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
fly launch --no-deploy --copy-config       # picks up fly.toml
fly secrets set GEMINI_API_KEY=...         # and NCBI_API_KEY if you have one
fly secrets set FRONTEND_ORIGINS=https://your-frontend.vercel.app
fly deploy
```

### Frontend → Vercel

1. Import the GitHub repo into Vercel; set the project root to `frontend/`.
2. Set environment variable `NEXT_PUBLIC_BACKEND_URL` to your Fly app URL (e.g. `https://cancer-to-hell-api.fly.dev`).
3. Deploy.
4. Add the Vercel URL to `FRONTEND_ORIGINS` on Fly so CORS allows it.

## API

### `POST /api/v1/decision-card`

Synchronous: runs all three agents and returns one JSON payload.

```bash
curl -s -X POST $BACKEND_URL/api/v1/decision-card \
  -H "Content-Type: application/json" \
  -d '{
    "cancer_type":"Metastatic Breast Cancer",
    "stage":"IV",
    "biomarkers":["HR+","HER2-","BRCA1"],
    "ecog":"1",
    "prior_lines":["Tamoxifen"],
    "labs":{"egfr":"89","liver_panel":"normal","cbc":"normal","ecg":"normal"}
  }'
```

### `POST /api/v1/decision-card/stream`

SSE: emits `status`, `module`, and `done` events as each agent finishes. Each `done` event includes any hallucinated PMIDs that were stripped from the output.

### `GET /health`

Liveness probe. Returns `200 {"status": "Cancer to Hell is running"}`.

## Project layout

```
backend/
  agents/         # specialist prompts and Gemini calls
  gemma/          # Gemini HTTP client
  handlers/       # HTTP handlers, citation validation, SSE
  pubmed/         # NCBI E-utilities client + structured-abstract parser
  main.go
  Dockerfile
  fly.toml
frontend/
  app/            # Next.js app router
```

## License

MIT
