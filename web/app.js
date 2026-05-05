const chat = document.getElementById("chat");
const form = document.getElementById("chatForm");
const input = document.getElementById("messageInput");
const collectionInput = document.getElementById("collectionInput");
const healthBtn = document.getElementById("healthBtn");
const collectionBtn = document.getElementById("collectionBtn");

let sessionID = localStorage.getItem("archimind_session_id");

if (!sessionID) {
  sessionID = crypto.randomUUID();
  localStorage.setItem("archimind_session_id", sessionID);
}

function addMessage(role, content, sources = []) {
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

function escapeHtml(str) {
  return String(str)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
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
    });

    loading.remove();
    addMessage("bot", data.answer, data.sources || []);
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
