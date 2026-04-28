"use client";

import { useMemo, useState } from "react";
import type * as React from "react";

type DecisionCardResponse = {
  status: string;
  message?: string;
  missing_data?: string[];
  decision_card?: {
    patient_snapshot: {
      cancer_type: string;
      stage: string;
      biomarkers: string[];
      ecog: string;
      prior_lines: string[];
    };
    top_regimens: Array<{
      name: string;
      rationale: string;
      evidence_grade: string;
      citations: string[];
    }>;
    safety_flags: string[];
    contraindications: string[];
    rationale_steps: string[];
    missing_data: string[];
    escalation: boolean;
    disclaimer: string;
  };
};

const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL ?? "http://localhost:8080";

export default function Home() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<DecisionCardResponse | null>(null);

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
    () => biomarkers.split(",").map((value) => value.trim()).filter(Boolean),
    [biomarkers],
  );

  async function onSubmit(event: React.SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setResult(null);
    setLoading(true);

    const payload = {
      cancer_type: cancerType,
      stage,
      biomarkers: parsedBiomarkers,
      ecog,
      prior_lines: priorLines.split(",").map((value) => value.trim()).filter(Boolean),
      labs: {
        egfr,
        liver_panel: liverPanel,
        cbc,
        ecg,
      },
    };

    const response = await fetch(`${backendUrl}/api/v1/decision-card`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      setLoading(false);
      setError(`Request failed with ${response.status}`);
      return;
    }

    const data = (await response.json()) as DecisionCardResponse;
    setResult(data);
    setLoading(false);
  }

  return (
    <main className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-6 py-8">
      <header className="space-y-2">
        <h1 className="text-3xl font-bold text-zinc-900">Cancer to Hell</h1>
        <p className="text-zinc-600">
          Local oncology co-pilot MVP for metastatic breast cancer treatment decision support.
        </p>
      </header>

      <form className="grid gap-4 rounded-xl border border-zinc-200 bg-white p-5 shadow-sm" onSubmit={onSubmit}>
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Cancer type" value={cancerType} onChange={setCancerType} />
          <Field label="Stage" value={stage} onChange={setStage} />
          <Field label="Biomarkers (comma separated)" value={biomarkers} onChange={setBiomarkers} />
          <Field label="ECOG" value={ecog} onChange={setEcog} />
          <Field label="Prior lines (comma separated)" value={priorLines} onChange={setPriorLines} />
          <Field label="eGFR" value={egfr} onChange={setEgfr} />
          <Field label="Liver panel" value={liverPanel} onChange={setLiverPanel} />
          <Field label="CBC" value={cbc} onChange={setCbc} />
          <Field label="ECG" value={ecg} onChange={setEcg} />
        </div> 
        <button
          type="submit"
          className="inline-flex w-fit items-center rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
          disabled={loading}
        >
          {loading ? "Generating..." : "Generate decision card"}
        </button>
      </form>

      {error && <p className="rounded-lg bg-red-50 p-4 text-red-700">{error}</p>}

      {result?.status === "blocked" && (
        <section className="rounded-xl border border-amber-200 bg-amber-50 p-5">
          <h2 className="text-lg font-semibold text-amber-900">Safety pre-check blocked</h2>
          <p className="mt-1 text-amber-800">{result.message}</p>
          <ul className="mt-3 list-disc pl-6 text-amber-800">
            {result.missing_data?.map((item) => <li key={item}>{item}</li>)}
          </ul>
        </section>
      )}

      {result?.status === "ok" && result.decision_card && (
        <section className="rounded-xl border border-zinc-200 bg-white p-5 shadow-sm">
          <h2 className="text-xl font-semibold text-zinc-900">Decision Card</h2>
          <p className="mt-1 text-sm text-zinc-600">{result.decision_card.disclaimer}</p>

          <h3 className="mt-4 text-sm font-semibold uppercase tracking-wide text-zinc-600">Top regimens</h3>
          <ul className="mt-2 space-y-3">
            {result.decision_card.top_regimens.map((regimen) => (
              <li key={regimen.name} className="rounded-lg border border-zinc-200 p-3">
                <p className="font-medium text-zinc-900">{regimen.name}</p>
                <p className="text-zinc-700">{regimen.rationale}</p>
                <p className="mt-1 text-sm text-zinc-600">Evidence: {regimen.evidence_grade}</p>
                <p className="text-sm text-zinc-600">Citations: {regimen.citations.join(", ")}</p>
              </li>
            ))}
          </ul>

          <h3 className="mt-4 text-sm font-semibold uppercase tracking-wide text-zinc-600">Safety flags</h3>
          <ul className="mt-2 list-disc pl-6 text-zinc-700">
            {result.decision_card.safety_flags.map((flag) => <li key={flag}>{flag}</li>)}
          </ul>
        </section>
      )}
    </main>
  );
}

function Field({ label, value, onChange }: Readonly<{ label: string; value: string; onChange: (value: string) => void }>) {
  return (
    <label className="flex flex-col gap-1 text-sm font-medium text-zinc-700">
      {label}
      <input
        className="rounded-md border border-zinc-300 px-3 py-2 text-sm text-zinc-900 outline-none ring-zinc-900/10 focus:ring"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}
