const chat = document.getElementById("chat");
const form = document.getElementById("chatForm");
const input = document.getElementById("messageInput");
const collectionInput = document.getElementById("collectionInput");
const compareCollectionInput = document.getElementById("compareCollectionInput");
const modeInput = document.getElementById("modeInput");
const modelInput = document.getElementById("modelInput");
const healthBtn = document.getElementById("healthBtn");
const collectionBtn = document.getElementById("collectionBtn");
const reviewBtn = document.getElementById("reviewBtn");
const exportMdBtn = document.getElementById("exportMdBtn");
const exportJsonBtn = document.getElementById("exportJsonBtn");
const frameworkBtn = document.getElementById("frameworkBtn");
const compareBtn = document.getElementById("compareBtn");

let sessionID = localStorage.getItem("archimind_session_id");

if (!sessionID) {
  sessionID = crypto.randomUUID();
  localStorage.setItem("archimind_session_id", sessionID);
}

const collectionVectors = {}; // name -> [vectorName, ...]

async function loadCollections() {
  try {
    const resp = await fetch("/api/collections");
    const data = await resp.json();
    const cols = data.collections || [];

    [collectionInput, compareCollectionInput].forEach((sel, i) => {
      sel.innerHTML = "";
      const blank = document.createElement("option");
      blank.value = "";
      blank.textContent = i === 0 ? ".env default" : "— none —";
      sel.appendChild(blank);
      cols.forEach(col => {
        collectionVectors[col.name] = col.vectors || [];
        const opt = document.createElement("option");
        opt.value = col.name;
        opt.textContent = col.name;
        sel.appendChild(opt);
      });
    });
  } catch (err) {
    [collectionInput, compareCollectionInput].forEach(sel => {
      sel.innerHTML = "<option value=''>Could not load</option>";
    });
  }
}

async function loadModels() {
  if (!modelInput) return;

  try {
    const response = await fetch("/api/models");
    const raw = await response.text();
    const data = raw ? JSON.parse(raw) : {};

    if (!response.ok) {
      throw new Error(data.error || "Could not load models");
    }

    const models = data.models || [];
    const active = String(data.active || "").trim();

    modelInput.innerHTML = "";
    models.forEach((model) => {
      const id = String(model.id || "").trim();
      if (!id) return;
      const opt = document.createElement("option");
      opt.value = id;
      opt.textContent = String(model.name || id);
      modelInput.appendChild(opt);
    });

    if (active) {
      const exists = models.some((model) => String(model.id || "").trim() === active);
      if (!exists) {
        const opt = document.createElement("option");
        opt.value = active;
        opt.textContent = active;
        modelInput.appendChild(opt);
      }
      modelInput.value = active;
    }

    if (modelInput.options.length === 0) {
      const opt = document.createElement("option");
      opt.value = "";
      opt.textContent = "No models available";
      modelInput.appendChild(opt);
    }
  } catch (err) {
    const fallback = String(modelInput.value || "").trim();
    modelInput.innerHTML = "";
    const opt = document.createElement("option");
    opt.value = fallback;
    opt.textContent = fallback || "Model list unavailable";
    modelInput.appendChild(opt);
  }
}

loadCollections();
loadModels();

// ── Preset questions per collection ──────────────────────

const PRESETS = {
  meta_reflections: [
    "What recurring concepts connect the Ra Material and Dolores Cannon reflections?",
    "Where do the Law of One teachings and Nostradamus channelings agree or contradict each other?",
    "What does this archive say about consciousness, free will, and the nature of spiritual evolution?",
    "What are the strongest claims about hidden history or suppressed knowledge?",
    "What echoes connect these esoteric teachings to older philosophical traditions?",
    "What questions does this archive raise that it cannot definitively answer?",
    "Summarize the model of reality that emerges across these reflections.",
    "What does this archive reveal about the relationship between physical health and spiritual service?",
    "Which source — Ra Material or Dolores Cannon — carries the most weight on the topic of consciousness?",
    "What patterns appear in the 'echoes' field across reflections — what older traditions keep surfacing?",
  ],
  mb_claims: [
    "What are the most strongly supported claims in this archive?",
    "What claims about consciousness or reality appear most frequently?",
    "What contradictions exist between competing claims?",
    "Which source texts generate the most high-confidence claims?",
    "What claims relate to hidden history or suppressed knowledge?",
    "Summarize the worldview that emerges from the strongest claims here.",
  ],
  mb_chunks: [
    "What are the dominant themes across these source texts?",
    "What passages describe the mechanics of consciousness?",
    "Where do the source texts agree on the nature of reality?",
    "What recurring metaphors appear across the archive?",
    "Summarize the core teachings found in this archive.",
  ],
  meistro_brain: [
    "What dominant themes appear across this knowledge archive?",
    "What recurring patterns emerge across these conversations?",
    "What ideas show the most conceptual development over time?",
    "What are the strongest original insights in this archive?",
    "What contradictions or unresolved tensions exist here?",
  ],
  chatbridge_core: [
    "What philosophical positions emerge most frequently across these sessions?",
    "Summarize the key debates from AI-to-AI discussion sessions.",
    "What consensus — if any — emerged across different AI models?",
    "What ideas were most challenged or contested in these sessions?",
    "What original insights appeared that surprised or pushed the discussion forward?",
  ],
  vectoreology_findings: [
    "What topology patterns were detected in the knowledge graph?",
    "What anomalies or unexpected clusters were found?",
    "Describe the cluster structure and what it reveals about the archive.",
    "What are the most isolated concepts in the vector space?",
    "What cross-domain connections were detected?",
  ],
  mb_sources: [
    "What source materials are represented in this archive?",
    "Which sources have the most coverage in the collection?",
    "What is the range of topics covered by the ingested sources?",
  ],
};

const DEFAULT_PRESETS = [
  "What are the strongest recurring themes in this archive?",
  "What contradictions exist in the retrieved material?",
  "Summarize the worldview that emerges from this collection.",
  "What original ideas appear here that don't echo elsewhere?",
  "What questions does this archive raise but not answer?",
];

const presetsEl = document.getElementById("presets");

function renderPresets(collectionName) {
  if (!presetsEl) return;
  const questions = PRESETS[collectionName] || (collectionName ? DEFAULT_PRESETS : []);
  presetsEl.innerHTML = "";
  questions.forEach(q => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "preset-chip";
    btn.textContent = q;
    btn.addEventListener("click", () => {
      input.value = q;
      input.focus();
    });
    presetsEl.appendChild(btn);
  });
}

const vectorInput = document.getElementById("vectorInput");
const vectorLabel = document.getElementById("vectorLabel");

// Preferred vector order for meta-bridge collections.
// claims_vec carries the full structured reflection (claims, echoes, questions).
// summary_vec is the compressed view. We prefer claims_vec for Q&A.
const VECTOR_PREFERENCE = ["claims_vec", "summary_vec"];

function updateVectorSelect(collectionName) {
  const vectors = collectionVectors[collectionName] || [];
  vectorInput.innerHTML = "";
  if (vectors.length === 0) {
    vectorLabel.style.display = "none";
    return;
  }
  vectorLabel.style.display = "";

  // Sort: preferred vectors first, then alphabetical.
  const sorted = [...vectors].sort((a, b) => {
    const ai = VECTOR_PREFERENCE.indexOf(a);
    const bi = VECTOR_PREFERENCE.indexOf(b);
    if (ai !== -1 && bi !== -1) return ai - bi;
    if (ai !== -1) return -1;
    if (bi !== -1) return 1;
    return a.localeCompare(b);
  });

  sorted.forEach((v, i) => {
    const opt = document.createElement("option");
    opt.value = v;
    opt.textContent = v;
    if (i === 0) opt.selected = true;
    vectorInput.appendChild(opt);
  });
}

collectionInput.addEventListener("change", () => {
  updateVectorSelect(collectionInput.value);
  renderPresets(collectionInput.value);
});

modelInput.addEventListener("change", async () => {
  const nextModel = modelInput.value.trim();
  if (!nextModel) return;

  const selectedName = modelInput.options[modelInput.selectedIndex]?.textContent || nextModel;

  try {
    const data = await postJSON("/api/model", { model: nextModel });
    if (data.model && data.model !== nextModel) {
      modelInput.value = data.model;
    }
    addMessage("bot", `Model switched to: ${selectedName}`);
  } catch (err) {
    addMessage("bot", `Model switch error: ${err.message}`);
    loadModels();
  }
});

function addMessage(role, content, sources = [], themes = [], contradictions = [], diagnostics = null, sourceInfluence = [], strongClaims = []) {
  // hide the empty state placeholder on first message
  const empty = chat.querySelector('.chat-empty');
  if (empty) empty.remove();

  const el = document.createElement("div");
  el.className = `message ${role}`;

  const label = document.createElement("div");
  label.className = "label";
  label.textContent = role === "user" ? "You" : "ArchiMind";

  const header = document.createElement("div");
  header.className = "message-header";
  header.appendChild(label);

  if (role === "bot") {
    const copyButton = document.createElement("button");
    copyButton.type = "button";
    copyButton.className = "message-copy-btn";
    copyButton.textContent = "Copy";
    copyButton.addEventListener("click", async () => {
      const copied = await copyText(buildCopyText(content, sources));
      copyButton.textContent = copied ? "Copied" : "Failed";
      setTimeout(() => {
        copyButton.textContent = "Copy";
      }, 1200);
    });
    header.appendChild(copyButton);
  }

  const body = document.createElement("div");
  body.className = "body markdown-body";
  body.innerHTML = (typeof marked !== 'undefined') ? marked.parse(content) : escapeHtml(content);

  el.appendChild(header);
  el.appendChild(body);

  if (themes.length > 0) {
    const themeWrap = document.createElement("div");
    themeWrap.className = "themes";

    const title = document.createElement("div");
    title.className = "themes-title";
    title.textContent = "Recurring themes";
    themeWrap.appendChild(title);

    const list = document.createElement("div");
    list.className = "theme-list";
    themes.forEach((theme) => {
      const chip = document.createElement("span");
      chip.className = "theme-chip";
      chip.textContent = `${theme.label} (${theme.count})`;
      list.appendChild(chip);
    });

    themeWrap.appendChild(list);
    el.appendChild(themeWrap);
  }

  if (contradictions.length > 0) {
    const contradictionWrap = document.createElement("div");
    contradictionWrap.className = "contradictions";

    const title = document.createElement("div");
    title.className = "contradictions-title";
    title.textContent = "Potential contradictions";
    contradictionWrap.appendChild(title);

    const list = document.createElement("ul");
    list.className = "contradiction-list";
    contradictions.forEach((entry) => {
      const item = document.createElement("li");
      item.textContent = `${entry.topic}: +${entry.supporting} / -${entry.opposing} (sources ${entry.mentioned_in})`;
      list.appendChild(item);
    });

    contradictionWrap.appendChild(list);
    el.appendChild(contradictionWrap);
  }

  if (strongClaims.length > 0) {
    const claimsWrap = document.createElement("div");
    claimsWrap.className = "diagnostics";
    const topClaims = strongClaims.slice(0, 3).map((claim, idx) => `${idx + 1}) ${claim.text} [c=${Number(claim.confidence || 0).toFixed(2)}]`).join(" | ");
    claimsWrap.textContent = `Strongest claims: ${topClaims}`;
    el.appendChild(claimsWrap);
  }

  if (sourceInfluence.length > 0) {
    const influenceWrap = document.createElement("div");
    influenceWrap.className = "diagnostics";
    const top = sourceInfluence.slice(0, 3).map((entry) => `[#${entry.index}] ${entry.title} (${Number(entry.influence || 0).toFixed(2)})`).join(" | ");
    influenceWrap.textContent = `Top source influence: ${top}`;
    el.appendChild(influenceWrap);
  }

  if (diagnostics) {
    const diagnosticWrap = document.createElement("div");
    diagnosticWrap.className = "diagnostics";
    diagnosticWrap.textContent = formatDiagnostics(diagnostics);
    el.appendChild(diagnosticWrap);
  }

  if (sources.length > 0) {
    const srcWrap = document.createElement("details");
    srcWrap.className = "sources";

    const summary = document.createElement("summary");
    summary.textContent = `Sources (${sources.length})`;
    srcWrap.appendChild(summary);

    sources.forEach((src) => {
      const srcEl = document.createElement("div");
      srcEl.className = "source";
      srcEl.innerHTML = `
        <strong>[${src.index}] ${escapeHtml(src.title || "Source")}</strong>
        <div>Score: ${Number(src.score || 0).toFixed(4)}</div>
        <pre>${escapeHtml(src.text || "")}</pre>
      `;
      srcWrap.appendChild(srcEl);
    });

    el.appendChild(srcWrap);
  }

  chat.appendChild(el);
  chat.scrollTop = chat.scrollHeight;
}

function addCompareSummary(result) {
  const left = result.left || {};
  const right = result.right || {};
  const leftThemes = (left.themes || []).map((t) => `${t.label} (${t.count})`).join(", ");
  const rightThemes = (right.themes || []).map((t) => `${t.label} (${t.count})`).join(", ");

  return [
    `Comparison: ${left.collection || "left"} vs ${right.collection || "right"}`,
    leftThemes ? `Left themes: ${leftThemes}` : "Left themes: none",
    rightThemes ? `Right themes: ${rightThemes}` : "Right themes: none",
  ].join("\n");
}

function buildCopyText(content, sources) {
  if (!sources || sources.length === 0) {
    return String(content || "");
  }

  const sourceText = sources.map((src) => {
    const title = src.title || "Source";
    const score = Number(src.score || 0).toFixed(4);
    const text = src.text || "";
    return `[${src.index}] ${title}\nScore: ${score}\n${text}`;
  }).join("\n\n");

  return `${String(content || "")}\n\nSources (${sources.length})\n${sourceText}`;
}

async function copyText(text) {
  const value = String(text || "");

  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(value);
      return true;
    } catch {
    }
  }

  const fallback = document.createElement("textarea");
  fallback.value = value;
  fallback.setAttribute("readonly", "");
  fallback.style.position = "fixed";
  fallback.style.opacity = "0";
  document.body.appendChild(fallback);
  fallback.select();
  const copied = document.execCommand("copy");
  fallback.remove();
  return copied;
}

function escapeHtml(str) {
  return String(str)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function formatDiagnostics(diagnostics) {
  const signals = diagnostics.unsupported_signals || [];
  const sample = signals.length > 0 ? ` Signals: ${signals.slice(0, 2).join(" | ")}` : "";
  return `Discipline: grounded=${diagnostics.grounded_claims || 0}, speculative=${diagnostics.speculative_claims || 0}, unsupported=${diagnostics.unsupported_claims || 0}, leap risk=${diagnostics.unsupported_leap_risk || "low"}.${sample}`;
}

function triggerDownload(blob, filename) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

async function postJSON(url, payload) {
  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  const raw = await response.text();

  if (!response.ok) {
    try {
      const parsed = JSON.parse(raw);
      throw new Error(parsed.error || raw);
    } catch {
      throw new Error(raw);
    }
  }

  return JSON.parse(raw);
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();

  const message = input.value.trim();
  if (!message) return;

  input.value = "";
  addMessage("user", message);

  const loading = document.createElement("div");
  loading.className = "message bot loading";
  loading.innerHTML = `<div class="label">ArchiMind</div><div class="body">Searching the archive...</div>`;
  chat.appendChild(loading);
  chat.scrollTop = chat.scrollHeight;

  try {
    const data = await postJSON("/api/chat", {
      session_id: sessionID,
      message,
      collection: collectionInput.value.trim(),
      vector_name: vectorInput.value.trim(),
      mode: modeInput.value,
    });

    loading.remove();
    addMessage("bot", data.answer, data.sources || [], data.themes || [], data.contradictions || [], data.diagnostics || null, data.source_influence || [], data.strong_claims || []);
  } catch (err) {
    loading.remove();
    addMessage("bot", `Error: ${err.message}`);
  }
});

healthBtn.addEventListener("click", async () => {
  try {
    const response = await fetch("/api/health");
    const data = await response.json();
    addMessage("bot", `Health:\n${JSON.stringify(data, null, 2)}`);
  } catch (err) {
    addMessage("bot", `Health check error: ${err.message}`);
  }
});

frameworkBtn.addEventListener("click", async () => {
  const message = input.value.trim();
  const collection = collectionInput.value.trim();

  if (!message) {
    addMessage("bot", "Framework needs a message.");
    return;
  }

  const collectionLabel = collection || ".env default";
  addMessage("user", `Framework from ${collectionLabel}: ${message}`);

  const loading = document.createElement("div");
  loading.className = "message bot loading";
  loading.innerHTML = `<div class="label">ArchiMind</div><div class="body">Extracting framework...</div>`;
  chat.appendChild(loading);
  chat.scrollTop = chat.scrollHeight;

  try {
    const data = await postJSON("/api/framework", {
      session_id: sessionID,
      message,
      collection,
      vector_name: vectorInput.value.trim(),
    });

    loading.remove();
    const components = (data.components || [])
      .map((component, index) => `${index + 1}. ${component.name}: ${component.principle}`)
      .join("\n");
    const text = [data.summary || "Framework extracted.", components ? `Components:\n${components}` : ""]
      .filter(Boolean)
      .join("\n\n");

    addMessage("bot", text, data.sources || [], data.themes || [], data.contradictions || [], null, data.source_influence || [], data.strong_claims || []);
  } catch (err) {
    loading.remove();
    addMessage("bot", `Framework error: ${err.message}`);
  }
});

compareBtn.addEventListener("click", async () => {
  const message = input.value.trim();
  const leftCollection = collectionInput.value.trim();
  const rightCollection = compareCollectionInput.value.trim();

  if (!message || !leftCollection || !rightCollection) {
    addMessage("bot", "Compare needs message, collection, and compare collection.");
    return;
  }

  addMessage("user", `Compare ${leftCollection} vs ${rightCollection}: ${message}`);

  const loading = document.createElement("div");
  loading.className = "message bot loading";
  loading.innerHTML = `<div class="label">ArchiMind</div><div class="body">Comparing collections...</div>`;
  chat.appendChild(loading);
  chat.scrollTop = chat.scrollHeight;

  try {
    const data = await postJSON("/api/compare", {
      session_id: sessionID,
      message,
      left_collection: leftCollection,
      right_collection: rightCollection,
      vector_name: vectorInput.value.trim(),
      mode: modeInput.value,
    });

    loading.remove();
    addMessage("bot", data.answer, [], []);

    const left = data.left || {};
    const right = data.right || {};
    const combinedSources = [...(left.sources || []), ...(right.sources || [])];
    const combinedThemes = [...(left.themes || []), ...(right.themes || [])];
    const combinedContradictions = [...(left.contradictions || []), ...(right.contradictions || [])];
    const combinedInfluence = [...(left.source_influence || []), ...(right.source_influence || [])]
      .sort((a, b) => Number(b.influence || 0) - Number(a.influence || 0))
      .slice(0, 5);
    const combinedClaims = [...(left.strong_claims || []), ...(right.strong_claims || [])]
      .sort((a, b) => Number(b.confidence || 0) - Number(a.confidence || 0))
      .slice(0, 5);
    addMessage("bot", addCompareSummary(data), combinedSources, combinedThemes, combinedContradictions, null, combinedInfluence, combinedClaims);
  } catch (err) {
    loading.remove();
    addMessage("bot", `Compare error: ${err.message}`);
  }
});

reviewBtn.addEventListener("click", async () => {
  try {
    const data = await postJSON("/api/review/last", { session_id: sessionID });
    const checklist = (data.diagnostics?.self_audit_checklist || []).map((line) => `- ${line}`).join("\n");
    const text = [
      "Last answer review",
      data.last_user_message ? `Question: ${data.last_user_message}` : "Question: (unknown)",
      formatDiagnostics(data.diagnostics || {}),
      checklist ? `Checklist:\n${checklist}` : "",
    ].filter(Boolean).join("\n\n");
    addMessage("bot", text);
  } catch (err) {
    addMessage("bot", `Review error: ${err.message}`);
  }
});

exportMdBtn.addEventListener("click", async () => {
  try {
    const response = await fetch("/api/export/markdown", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session_id: sessionID }),
    });

    if (!response.ok) {
      const raw = await response.text();
      throw new Error(raw);
    }

    const blob = await response.blob();
    triggerDownload(blob, "archimind_chat_export.md");
    addMessage("bot", "Markdown export downloaded.");
  } catch (err) {
    addMessage("bot", `Export markdown error: ${err.message}`);
  }
});

exportJsonBtn.addEventListener("click", async () => {
  try {
    const response = await fetch("/api/export/json", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session_id: sessionID }),
    });

    if (!response.ok) {
      const raw = await response.text();
      throw new Error(raw);
    }

    const blob = await response.blob();
    triggerDownload(blob, "archimind_chat_export.json");
    addMessage("bot", "JSON export downloaded.");
  } catch (err) {
    addMessage("bot", `Export JSON error: ${err.message}`);
  }
});

collectionBtn.addEventListener("click", async () => {
  try {
    const name = collectionInput.value.trim();
    const url = name ? `/api/collection?name=${encodeURIComponent(name)}` : "/api/collection";
    const response = await fetch(url);
    const raw = await response.text();

    if (!response.ok) {
      try {
        const parsed = JSON.parse(raw);
        throw new Error(parsed.error || raw);
      } catch {
        throw new Error(raw);
      }
    }

    const data = JSON.parse(raw);
    addMessage("bot", `Collection info:\n${JSON.stringify(data, null, 2)}`);
  } catch (err) {
    addMessage("bot", `Collection check error: ${err.message}`);
  }
});
