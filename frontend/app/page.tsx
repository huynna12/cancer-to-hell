"use client";

import { useMemo, useState } from "react";

type DebateResponse = {
  status: string;
  message?: string;
  missing_data?: string[];
  evidence?: string;
  guideline?: string;
  safety?: string;
};

const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL ?? "http://localhost:8080";

export default function Home() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<DebateResponse | null>(null);

  const [cancerType, setCancerType] = useState("Metastatic Breast Cancer");
  const [stage, setStage] = useState("IV");
  const [biomarkers, setBiomarkers] = useState("HR+,HER2-,BRCA1");
  const [ecog, setEcog] = useState("1");
  const [priorLines, setPriorLines] = useState("Tamoxifen");
  const [egfr, setEgfr] = useState("");
  const [liverPanel, setLiverPanel] = useState("");
  const [cbc, setCbc] = useState("");
  const [ecg, setEcg] = useState("");

  const parsedBiomarkers = useMemo(
    () => biomarkers.split(",").map((v) => v.trim()).filter(Boolean),
    [biomarkers]
  );

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setResult(null);
    setLoading(true);

    const payload = {
      cancer_type: cancerType,
      stage,
      biomarkers: parsedBiomarkers,
      ecog,
      prior_lines: priorLines.split(",").map((v) => v.trim()).filter(Boolean),
      labs: { egfr, liver_panel: liverPanel, cbc, ecg },
    };

    try {
      const response = await fetch(`${backendUrl}/api/v1/decision-card`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        setError(`Request failed with ${response.status}`);
        return;
      }

      const data = (await response.json()) as DebateResponse;
      setResult(data);
    } catch {
      setError("Failed to connect to backend. Make sure the server is running.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="min-h-screen" style={{ backgroundColor: "#EDE9E6" }}>
      <div className="mx-auto max-w-6xl px-6 py-10">

        {/* Header */}
        <header className="mb-8">
          <div className="flex items-center gap-3 mb-2">
            <div
              className="w-10 h-10 rounded-xl flex items-center justify-center text-white font-bold text-lg"
              style={{ backgroundColor: "#C9996B" }}
            >
              C
            </div>
            <h1 className="text-4xl font-bold" style={{ color: "#5C4F4A" }}>
              Cancer to Hell
            </h1>
          </div>
          <p className="text-base font-medium" style={{ color: "#5C4F4A" }}>
            Local oncology co-pilot for metastatic breast cancer tumor board decisions.
          </p>
          <p className="mt-1 text-sm" style={{ color: "#5C766D" }}>
            🔒 Powered by Gemma 4 — running entirely on-device. Patient data never leaves this machine.
          </p>
        </header>

        {/* Form */}
        <form
          onSubmit={onSubmit}
          className="rounded-2xl p-6 shadow-sm mb-6"
          style={{ backgroundColor: "white", border: "1px solid #C9996B" }}
        >
          <h2 className="text-lg font-bold mb-1" style={{ color: "#5C4F4A" }}>
            Patient Profile
          </h2>
          <p className="text-xs mb-4" style={{ color: "#5C766D" }}>
            All fields marked with labs are required before a recommendation can be generated.
          </p>

          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Cancer Type" value={cancerType} onChange={setCancerType} />
            <Field label="Stage" value={stage} onChange={setStage} />
            <Field label="Biomarkers (comma separated)" value={biomarkers} onChange={setBiomarkers} />
            <Field label="ECOG Performance Status *" value={ecog} onChange={setEcog} />
            <Field label="Prior Lines (comma separated)" value={priorLines} onChange={setPriorLines} />
            <Field label="eGFR — kidney function *" value={egfr} onChange={setEgfr} />
            <Field label="Liver Panel *" value={liverPanel} onChange={setLiverPanel} />
            <Field label="CBC — complete blood count *" value={cbc} onChange={setCbc} />
            <Field label="ECG" value={ecg} onChange={setEcg} />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="mt-6 px-6 py-3 rounded-xl text-white font-semibold text-sm transition-opacity disabled:opacity-60 w-full md:w-auto"
            style={{ backgroundColor: loading ? "#5C766D" : "#C9996B" }}
          >
            {loading ? "⏳ Generating — this may take around 5 minutes..." : "Generate Decision Card →"}
          </button>
        </form>

        {/* Error */}
        {error && (
          <div
            className="rounded-xl p-4 mb-6"
            style={{ backgroundColor: "#fdf0f0", border: "1px solid #e57373" }}
          >
            <p className="text-sm font-medium" style={{ color: "#c62828" }}>
              ❌ {error}
            </p>
          </div>
        )}

        {/* Safety Blocked */}
        {result?.status === "blocked" && (
          <div
            className="rounded-2xl p-6 mb-6"
            style={{ backgroundColor: "#fff8e1", border: "2px solid #C9996B" }}
          >
            <h2 className="text-lg font-bold" style={{ color: "#5C4F4A" }}>
              ⚠️ Safety Pre-Check Blocked
            </h2>
            <p className="mt-1 text-sm" style={{ color: "#5C4F4A" }}>
              {result.message}
            </p>
            <ul className="mt-3 list-disc pl-6 text-sm space-y-1" style={{ color: "#5C4F4A" }}>
              {result.missing_data?.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>
        )}

        {/* Decision Card */}
        {result?.status === "ok" && (
          <div className="space-y-6">

            {/* Section label */}
            <div className="flex items-center gap-2">
              <div className="h-px flex-1" style={{ backgroundColor: "#C9996B" }} />
              <span className="text-sm font-semibold px-3" style={{ color: "#5C4F4A" }}>
                Decision Card
              </span>
              <div className="h-px flex-1" style={{ backgroundColor: "#C9996B" }} />
            </div>

            {/* 3 Module Cards — stacked vertically for readability */}
            <div className="flex flex-col gap-6">
              <ModuleCard
                emoji="🔬"
                title="Evidence Retrieval"
                content={result.evidence ?? ""}
                accentColor="#C9996B"
              />
              <ModuleCard
                emoji="📋"
                title="Guideline Alignment"
                content={result.guideline ?? ""}
                accentColor="#5C766D"
              />
              <ModuleCard
                emoji="⚠️"
                title="Safety & Risk"
                content={result.safety ?? ""}
                accentColor="#5C4F4A"
              />
            </div>

{/* Disclaimer */}
            <div
              className="rounded-xl p-4 text-center"
              style={{ backgroundColor: "white", border: "1px solid #5C766D" }}
            >
              <p className="text-xs" style={{ color: "#5C766D" }}>
                ⚕️ Clinical decision support only. Not a substitute for physician judgment.
                All recommendations must be reviewed by a qualified oncologist before clinical use.
              </p>
            </div>

          </div>
        )}
      </div>
    </main>
  );
}

function Field({
  label,
  value,
  onChange,
}: Readonly<{
  label: string;
  value: string;
  onChange: (value: string) => void;
}>) {
  return (
    <label className="flex flex-col gap-1 text-sm font-medium" style={{ color: "#5C4F4A" }}>
      {label}
      <input
        className="rounded-lg px-3 py-2 text-sm outline-none focus:ring-2"
        style={{
          border: "1px solid #C9996B",
          backgroundColor: "#EDE9E6",
          color: "#5C4F4A",
        }}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </label>
  );
}

// Splits text on "PMID: 12345678" and turns each match into a PubMed link
function renderWithLinks(text: string, linkColor: string) {
  const parts = text.split(/(PMID:\s*\d+)/g);
  return parts.map((part, i) => {
    const match = part.match(/PMID:\s*(\d+)/);
    if (match) {
      const pmid = match[1];
      return (
        <a
          key={i}
          href={`https://pubmed.ncbi.nlm.nih.gov/${pmid}`}
          target="_blank"
          rel="noopener noreferrer"
          className="underline underline-offset-2 font-medium hover:opacity-70 transition-opacity"
          style={{ color: linkColor }}
        >
          {part}
        </a>
      );
    }
    return <span key={i}>{part}</span>;
  });
}

function ModuleCard({
  emoji,
  title,
  content,
  accentColor,
}: Readonly<{
  emoji: string;
  title: string;
  content: string;
  accentColor: string;
}>) {
  return (
    <div
      className="rounded-2xl overflow-hidden flex"
      style={{ backgroundColor: "white", border: "1px solid #E0D9D4" }}
    >
      {/* Left accent bar */}
      <div className="w-1.5 shrink-0" style={{ backgroundColor: accentColor }} />

      <div className="flex flex-col gap-3 p-6 w-full">
        {/* Header */}
        <div className="flex items-center gap-2">
          <span className="text-2xl">{emoji}</span>
          <h3 className="font-bold text-base" style={{ color: accentColor }}>
            {title}
          </h3>
        </div>
        {/* Content — PMIDs rendered as clickable PubMed links */}
        <div
          className="text-sm leading-7 whitespace-pre-wrap"
          style={{ color: "#5C4F4A" }}
        >
          {renderWithLinks(content, accentColor)}
        </div>
      </div>
    </div>
  );
}