const chat = document.getElementById("chat");
const form = document.getElementById("chatForm");
const input = document.getElementById("messageInput");
const collectionInput = document.getElementById("collectionInput");
const compareCollectionInput = document.getElementById("compareCollectionInput");
const modeInput = document.getElementById("modeInput");
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

function addMessage(role, content, sources = [], themes = [], contradictions = [], diagnostics = null, sourceInfluence = [], strongClaims = []) {
  const el = document.createElement("div");
  el.className = `message ${role}`;

  const label = document.createElement("div");
  label.className = "label";
  label.textContent = role === "user" ? "You" : "ArchiMind";

  const body = document.createElement("div");
  body.className = "body";
  body.textContent = content;

  el.appendChild(label);
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

  if (!message || !collection) {
    addMessage("bot", "Framework needs message and collection.");
    return;
  }

  addMessage("user", `Framework from ${collection}: ${message}`);

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
