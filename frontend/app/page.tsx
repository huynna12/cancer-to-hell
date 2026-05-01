"use client";

import { useMemo, useState } from "react";
import ReactMarkdown from "react-markdown";

type ModuleState = { content: string; done: boolean };

type AppState = {
  phase: "idle" | "working" | "done" | "blocked";
  statusMessage: string;
  evidence: ModuleState;
  guideline: ModuleState;
  safety: ModuleState;
  missing_data?: string[];
  blockMessage?: string;
  hallucinatedCitations?: string[];
  error?: string;
};

const IDLE: AppState = {
  phase: "idle",
  statusMessage: "",
  evidence: { content: "", done: false },
  guideline: { content: "", done: false },
  safety: { content: "", done: false },
};

const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL ?? "http://localhost:8080";

// ── Validation ────────────────────────────────────────────────────────────────

function validateForm(ecog: string, egfr: string): string | null {
  const ecogNum = Number(ecog);
  if (ecog.trim() === "" || !Number.isInteger(ecogNum) || ecogNum < 0 || ecogNum > 4) {
    return "ECOG must be an integer between 0 and 4.";
  }
  if (egfr.trim() !== "" && (isNaN(Number(egfr)) || Number(egfr) <= 0)) {
    return "eGFR must be a positive number (e.g. 62).";
  }
  return null;
}

// Converts "PMID: 12345678" in model output into markdown links
function injectPubMedLinks(text: string): string {
  return text.replace(
    /PMID:\s*(\d+)/g,
    (_, pmid) => `[PMID: ${pmid}](https://pubmed.ncbi.nlm.nih.gov/${pmid})`
  );
}

// ── Main component ────────────────────────────────────────────────────────────

export default function Home() {
  const [state, setState] = useState<AppState>(IDLE);
  const [validationError, setValidationError] = useState<string | null>(null);

  const [cancerType, setCancerType] = useState("Metastatic Breast Cancer");
  const [stage, setStage] = useState("IV");
  const [biomarkers, setBiomarkers] = useState("HR+,HER2-,BRCA1");
  const [ecog, setEcog] = useState("1");
  const [priorLines, setPriorLines] = useState("Tamoxifen");
  const [comorbidities, setComorbidities] = useState("");
  const [currentMeds, setCurrentMeds] = useState("");
  const [egfr, setEgfr] = useState("");
  const [liverPanel, setLiverPanel] = useState("");
  const [cbc, setCbc] = useState("");
  const [ecg, setEcg] = useState("");

  const parsedBiomarkers = useMemo(
    () => biomarkers.split(",").map((v) => v.trim()).filter(Boolean),
    [biomarkers]
  );

  const isWorking = state.phase === "working";

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setValidationError(null);

    const err = validateForm(ecog, egfr);
    if (err) {
      setValidationError(err);
      return;
    }

    setState({
      phase: "working",
      statusMessage: "Connecting...",
      evidence: { content: "", done: false },
      guideline: { content: "", done: false },
      safety: { content: "", done: false },
    });

    const payload = {
      cancer_type: cancerType,
      stage,
      biomarkers: parsedBiomarkers,
      ecog,
      prior_lines: priorLines.split(",").map((v) => v.trim()).filter(Boolean),
      comorbidities: comorbidities.split(",").map((v) => v.trim()).filter(Boolean),
      current_meds: currentMeds.split(",").map((v) => v.trim()).filter(Boolean),
      labs: { egfr, liver_panel: liverPanel, cbc, ecg },
    };

    try {
      const response = await fetch(`${backendUrl}/api/v1/decision-card/stream`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      const contentType = response.headers.get("Content-Type") ?? "";
      if (!response.ok || contentType.includes("application/json")) {
        const data = await response.json();
        if (data.status === "blocked") {
          setState((s) => ({
            ...s,
            phase: "blocked",
            missing_data: data.missing_data,
            blockMessage: data.message,
          }));
        } else {
          setState((s) => ({ ...s, phase: "idle", error: data.error ?? "Unknown error" }));
        }
        return;
      }

      if (!response.body) throw new Error("No response body");

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const messages = buffer.split("\n\n");
        buffer = messages.pop() ?? "";

        for (const message of messages) {
          const dataLine = message.split("\n").find((l) => l.startsWith("data: "));
          if (!dataLine) continue;
          try {
            const evt = JSON.parse(dataLine.slice(6)) as {
              type: string;
              name?: string;
              content?: string;
              message?: string;
              missing_data?: string[];
              hallucinated_citations?: string[];
            };

            if (evt.type === "status") {
              setState((s) => ({ ...s, statusMessage: evt.message ?? "" }));
            } else if (evt.type === "module" && evt.name) {
              setState((s) => ({
                ...s,
                [evt.name!]: { content: evt.content ?? "", done: true },
              }));
            } else if (evt.type === "done") {
              setState((s) => ({
                ...s,
                phase: "done",
                hallucinatedCitations: evt.hallucinated_citations,
              }));
            } else if (evt.type === "blocked") {
              setState((s) => ({
                ...s,
                phase: "blocked",
                missing_data: evt.missing_data,
                blockMessage: evt.message,
              }));
            } else if (evt.type === "error") {
              setState((s) => ({ ...s, phase: "idle", error: evt.message }));
            }
          } catch {
            // malformed SSE line — skip
          }
        }
      }
    } catch {
      setState((s) => ({
        ...s,
        phase: "idle",
        error: "Failed to connect to backend. Make sure the server is running.",
      }));
    }
  }

  const showResults = state.phase === "working" || state.phase === "done";

  return (
    <main className="min-h-screen" style={{ backgroundColor: "#EDE9E6" }}>
      <div className="mx-auto max-w-6xl px-6 py-10">

        {/* Medical disclaimer */}
        <div
          role="alert"
          className="mb-6 rounded-xl px-4 py-3 text-sm"
          style={{ backgroundColor: "#FFF4E0", border: "1px solid #C9996B", color: "#5C4F4A" }}
        >
          <strong>Research and educational use only.</strong> This tool is a portfolio
          demonstration of multi-agent LLM orchestration. Outputs are not medical advice,
          have not been clinically validated, and must not be used to make treatment decisions.
        </div>

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
            Powered by Gemma 4 via Google AI Studio. Patient data never leaves your machine.
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
            Fields marked with * are required before a recommendation can be generated.
          </p>

          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Cancer Type" value={cancerType} onChange={setCancerType} />
            <Field label="Stage" value={stage} onChange={setStage} />
            <Field label="Biomarkers (comma separated)" value={biomarkers} onChange={setBiomarkers} />
            <Field
              label="ECOG Performance Status * (0–4)"
              value={ecog}
              onChange={setEcog}
              inputProps={{ type: "number", min: "0", max: "4", step: "1" }}
            />
            <Field label="Prior Lines (comma separated)" value={priorLines} onChange={setPriorLines} />
            <Field
              label="eGFR — kidney function * (numeric, e.g. 62)"
              value={egfr}
              onChange={setEgfr}
              inputProps={{ type: "number", min: "0", step: "any" }}
            />
            <Field label="Liver Panel *" value={liverPanel} onChange={setLiverPanel} />
            <Field label="CBC — complete blood count *" value={cbc} onChange={setCbc} />
            <Field label="ECG" value={ecg} onChange={setEcg} />
            <Field
              label="Comorbidities (comma separated)"
              value={comorbidities}
              onChange={setComorbidities}
            />
            <Field
              label="Current Medications (comma separated)"
              value={currentMeds}
              onChange={setCurrentMeds}
            />
          </div>

          {validationError && (
            <p className="mt-3 text-sm font-medium" style={{ color: "#c62828" }}>
              ⚠️ {validationError}
            </p>
          )}

          <button
            type="submit"
            disabled={isWorking}
            className="mt-6 px-6 py-3 rounded-xl text-white font-semibold text-sm transition-opacity disabled:opacity-60 w-full md:w-auto"
            style={{ backgroundColor: isWorking ? "#5C766D" : "#C9996B" }}
          >
            {isWorking
              ? `⏳ ${state.statusMessage || "Working..."}`
              : "Generate Decision Card →"}
          </button>
        </form>

        {/* Connection error */}
        {state.error && (
          <div
            className="rounded-xl p-4 mb-6"
            style={{ backgroundColor: "#fdf0f0", border: "1px solid #e57373" }}
          >
            <p className="text-sm font-medium" style={{ color: "#c62828" }}>
              ❌ {state.error}
            </p>
          </div>
        )}

        {/* Safety Blocked */}
        {state.phase === "blocked" && (
          <div
            className="rounded-2xl p-6 mb-6"
            style={{ backgroundColor: "#fff8e1", border: "2px solid #C9996B" }}
          >
            <h2 className="text-lg font-bold" style={{ color: "#5C4F4A" }}>
              ⚠️ Safety Pre-Check Blocked
            </h2>
            <p className="mt-1 text-sm" style={{ color: "#5C4F4A" }}>
              {state.blockMessage}
            </p>
            <ul className="mt-3 list-disc pl-6 text-sm space-y-1" style={{ color: "#5C4F4A" }}>
              {state.missing_data?.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>
        )}

        {/* Hallucinated citations warning */}
        {state.phase === "done" && state.hallucinatedCitations && state.hallucinatedCitations.length > 0 && (
          <div
            className="rounded-xl p-4 mb-4"
            style={{ backgroundColor: "#fff3e0", border: "1px solid #fb8c00" }}
          >
            <p className="text-sm font-medium" style={{ color: "#e65100" }}>
              ⚠️ {state.hallucinatedCitations.length} citation(s) could not be verified against retrieved PubMed papers and were marked [unverified].
            </p>
          </div>
        )}

        {/* Decision Card — streams in as modules arrive */}
        {showResults && (
          <div className="space-y-6">
            <div className="flex items-center gap-2">
              <div className="h-px flex-1" style={{ backgroundColor: "#C9996B" }} />
              <span className="text-sm font-semibold px-3" style={{ color: "#5C4F4A" }}>
                Decision Card
              </span>
              <div className="h-px flex-1" style={{ backgroundColor: "#C9996B" }} />
            </div>

            <div className="flex flex-col gap-6">
              <ModuleCard
                emoji="🔬"
                title="Evidence Retrieval"
                module={state.evidence}
                accentColor="#C9996B"
              />
              <ModuleCard
                emoji="📋"
                title="Guideline Alignment"
                module={state.guideline}
                accentColor="#5C766D"
              />
              <ModuleCard
                emoji="⚠️"
                title="Risk"
                module={state.safety}
                accentColor="#A65A4F"
              />
            </div>

            {state.phase === "done" && (
              <div
                className="rounded-xl p-4 text-center"
                style={{ backgroundColor: "white", border: "1px solid #5C766D" }}
              >
                <p className="text-xs" style={{ color: "#5C766D" }}>
                  ⚕️ Clinical decision support only. Not a substitute for physician judgment.
                  All recommendations must be reviewed by a qualified oncologist before clinical use.
                </p>
              </div>
            )}
          </div>
        )}
      </div>
    </main>
  );
}

// ── Field ─────────────────────────────────────────────────────────────────────

function Field({
  label,
  value,
  onChange,
  inputProps,
}: Readonly<{
  label: string;
  value: string;
  onChange: (value: string) => void;
  inputProps?: React.InputHTMLAttributes<HTMLInputElement>;
}>) {
  return (
    <label className="flex flex-col gap-1 text-sm font-medium" style={{ color: "#5C4F4A" }}>
      {label}
      <input
        className="rounded-lg px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-[#C9996B]"
        style={{ border: "1px solid #C9996B", backgroundColor: "#EDE9E6", color: "#5C4F4A" }}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        {...inputProps}
      />
    </label>
  );
}

// ── ModuleCard ────────────────────────────────────────────────────────────────

function ModuleCard({
  emoji,
  title,
  module,
  accentColor,
}: Readonly<{
  emoji: string;
  title: string;
  module: ModuleState;
  accentColor: string;
}>) {
  return (
    <div
      className="rounded-2xl overflow-hidden flex"
      style={{ backgroundColor: "white", border: "1px solid #E0D9D4" }}
    >
      <div className="w-1.5 shrink-0" style={{ backgroundColor: accentColor }} />
      <div className="flex flex-col gap-3 p-6 w-full">
        <div className="flex items-center gap-2">
          <span className="text-2xl">{emoji}</span>
          <h3 className="font-bold text-base" style={{ color: accentColor }}>{title}</h3>
          {!module.done && (
            <span className="ml-auto text-xs animate-pulse" style={{ color: "#5C766D" }}>
              generating...
            </span>
          )}
        </div>

        {module.done ? (
          <div className="prose-sm max-w-none">
            <ReactMarkdown
              components={{
                p: ({ children }) => (
                  <p className="text-sm leading-7 mb-3" style={{ color: "#5C4F4A" }}>{children}</p>
                ),
                strong: ({ children }) => (
                  <strong className="font-semibold" style={{ color: "#5C4F4A" }}>{children}</strong>
                ),
                em: ({ children }) => (
                  <em className="italic" style={{ color: "#5C4F4A" }}>{children}</em>
                ),
                ul: ({ children }) => (
                  <ul className="list-disc pl-5 space-y-1 mb-3 text-sm" style={{ color: "#5C4F4A" }}>{children}</ul>
                ),
                ol: ({ children }) => (
                  <ol className="list-decimal pl-5 space-y-1 mb-3 text-sm" style={{ color: "#5C4F4A" }}>{children}</ol>
                ),
                li: ({ children }) => (
                  <li className="leading-7" style={{ color: "#5C4F4A" }}>{children}</li>
                ),
                a: ({ href, children }) => (
                  <a
                    href={href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="underline underline-offset-2 font-medium hover:opacity-70 transition-opacity"
                    style={{ color: accentColor }}
                  >
                    {children}
                  </a>
                ),
              }}
            >
              {injectPubMedLinks(module.content)}
            </ReactMarkdown>
          </div>
        ) : (
          <div className="space-y-2 animate-pulse">
            <div className="h-3 rounded" style={{ backgroundColor: "#EDE9E6", width: "90%" }} />
            <div className="h-3 rounded" style={{ backgroundColor: "#EDE9E6", width: "75%" }} />
            <div className="h-3 rounded" style={{ backgroundColor: "#EDE9E6", width: "85%" }} />
            <div className="h-3 rounded" style={{ backgroundColor: "#EDE9E6", width: "60%" }} />
          </div>
        )}
      </div>
    </div>
  );
}
