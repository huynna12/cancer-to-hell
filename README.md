# Cancer to Hell

Local, privacy-preserving oncology co-pilot MVP for metastatic breast cancer tumor-board decisions.

## Current MVP status

This repository now contains:

- **Go + Gin backend** with:
  - `POST /api/v1/decision-card` safety pre-check + decision card JSON
  - `POST /api/v1/decision-card/stream` SSE progress events + final decision card
  - 3 concurrent clinical modules (simulated): evidence retrieval, guideline alignment, safety/risk
- **Next.js frontend** with:
  - structured patient input form
  - safety block display for missing critical data
  - decision card rendering for top regimens, evidence grade, citations, and safety flags

## Run locally

### 1. Backend

```bash
cd backend
go run .
```

Backend default: `http://localhost:8080`

### 2. Frontend

```bash
cd frontend
npm install
NEXT_PUBLIC_BACKEND_URL=http://localhost:8080 npm run dev
```

Frontend default: `http://localhost:3000`

## API example

```bash
curl -s -X POST http://localhost:8080/api/v1/decision-card \
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

## Important disclaimer

This software is for hackathon demonstration and clinical decision support prototyping only. It is **not** a medical device and must not be used for autonomous diagnosis or treatment decisions.