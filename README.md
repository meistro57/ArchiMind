# ArchiMind
<img width="800" height="800" alt="image" src="https://github.com/user-attachments/assets/d2b5f903-e36e-481f-a445-90cb38515b5a" />

ArchiMind is a Go-based retrieval cockpit for querying Qdrant collections with source-cited answers, retrieval diagnostics, and runtime model controls.

## Stack

- Go 1.22+
- Qdrant (vector retrieval)
- Redis (chat history + caches)
- OpenRouter (chat + optional embeddings)
- Ollama (optional embeddings)
- Static web UI (`web/`)

## Core capabilities

- Source-cited chat with diagnostics (`/api/chat`)
- Answer modes: `normal`, `skeptical`, `synthesis`, `diagnostic`
- Collection comparison (`/api/compare`)
- Framework extraction (`/api/framework`)
- Last-answer review (`/api/review/last`)
- Session export (`/api/export/markdown`, `/api/export/json`)
- Background report generation (`/api/report`)
- Runtime chat-model switching (`GET /api/models`, `POST /api/model`)
- Collection/vector discovery in UI (`GET /api/collections`)
- Assistant response copy-to-clipboard actions in the web UI
- Hybrid retrieval (dense + BM25 RRF) via `HYBRID_SEARCH`
- Meta-reflection fan-out retrieval (`meta_reflections` -> `mb_chunks`) for exact keyword recall
- Dead reflection filtering (`reflection_confidence == 0` or `is_empty_reflection == true`)
- Local Qdrant worker pool for faster retrieval job handling

## Retrieval behavior highlights

- `meta_reflections` queries fan out to `mb_chunks`, then merge/deduplicate/re-rank before prompting.
- Reflection points with zero confidence or empty-reflection flag are dropped before context assembly.
- Hybrid retrieval fuses dense ranks with BM25 lexical ranks when `HYBRID_SEARCH=true`.
- Worker pool executes Qdrant query jobs locally (goroutine pool + queue) to reduce request-path stalls.

## Runtime model switching
<img width="1074" height="1761" alt="image" src="https://github.com/user-attachments/assets/31d61c90-4fd8-4121-9b27-629910b544cb" />

## Examine Sources used

<img width="1089" height="1774" alt="image" src="https://github.com/user-attachments/assets/0d8f2340-af2a-40e3-9f7a-b13113b36ade" />

- The UI model dropdown loads options from `GET /api/models`.
- Changing the model calls `POST /api/model` and takes effect on the next chat request.
- The list is cached in Redis at key `openrouter:models`.
- Background report generation uses a dedicated worker model: `google/gemini-2.0-flash-001`.

## Quick start

1. Copy env template:

```bash
cp .env.example .env
```

2. Start local infra (Qdrant + Redis):

```bash
./scripts/docker/install_qdrant_redis.sh
```

3. Run app:

```bash
./scripts/rebuild-and-start.sh
```

4. Open:

```text
http://localhost:8090
```

## Environment

See `.env.example` for all supported keys. Minimum commonly-used keys:

```env
APP_PORT=8090
OPENROUTER_API_KEY=sk-or-...
OPENROUTER_MODEL=deepseek/deepseek-r1
QDRANT_URL=http://localhost:6333
QDRANT_COLLECTION=meta_reflections
QDRANT_VECTOR_NAME=claims_vec
QDRANT_TOP_K=8
EMBED_PROVIDER=openrouter
OPENROUTER_EMBED_MODEL=openai/text-embedding-3-small
REDIS_ADDR=localhost:6379
CACHE_EMBEDDINGS=true
CACHE_QDRANT_RESULTS=true
HYBRID_SEARCH=true
```

## HTTP API

- `POST /api/chat`
- `POST /api/compare`
- `POST /api/framework`
- `POST /api/review/last`
- `POST /api/export/markdown`
- `POST /api/export/json`
- `POST /api/report`
- `GET /api/health`
- `GET /api/collection`
- `GET /api/collections`
- `GET /api/models`
- `POST /api/model`

## API usage samples

The examples below assume ArchiMind is running locally at `http://localhost:8090`. Override the base URL with `ARCHIMIND_BASE_URL` when calling a remote instance.

Common request fields:

- `session_id`: Optional conversation key. Defaults to `default` when omitted.
- `collection`: Optional Qdrant collection override. Defaults to `QDRANT_COLLECTION`.
- `vector_name`: Optional vector override. If omitted, ArchiMind resolves the best available vector for the selected collection.
- `mode`: Optional answer mode. Supported common values are `normal`, `skeptical`, `synthesis`, and `diagnostic`.
- Dimension/vector mismatches now return HTTP 400 with structured mismatch details instead of a generic connectivity hint.

<details open>
<summary><strong>Python</strong></summary>

Save as `archimind_api_samples.py`, then run `python3 archimind_api_samples.py`.

```python
# archimind_api_samples.py
from __future__ import annotations

import json
import os
import sys
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

BASE_URL = os.getenv("ARCHIMIND_BASE_URL", "http://localhost:8090").rstrip("/")
SESSION_ID = os.getenv("ARCHIMIND_SESSION_ID", "readme-python-demo")


def request_json(method: str, path: str, payload: dict[str, Any] | None = None) -> dict[str, Any]:
    """Send a JSON request to ArchiMind and return the decoded JSON response."""
    body = json.dumps(payload).encode("utf-8") if payload is not None else None
    request = Request(
        f"{BASE_URL}{path}",
        data=body,
        method=method,
        headers={"Content-Type": "application/json", "Accept": "application/json"},
    )

    try:
        with urlopen(request, timeout=60) as response:
            return json.loads(response.read().decode("utf-8"))
    except HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"ArchiMind returned HTTP {exc.code}: {detail}") from exc
    except URLError as exc:
        raise RuntimeError(f"Could not reach ArchiMind at {BASE_URL}: {exc.reason}") from exc


def main() -> int:
    """Run a small tour of the public API."""
    try:
        health = request_json("GET", "/api/health")
        print(f"Health: {health['status']} ({health['app']} {health['app_version']})")

        chat = request_json(
            "POST",
            "/api/chat",
            {
                "session_id": SESSION_ID,
                "message": "Summarise the strongest retrieved evidence in three bullets.",
                "mode": "normal",
            },
        )
        print("\nAnswer:\n", chat.get("answer", ""))
        print("\nSources returned:", len(chat.get("sources", [])))

        models = request_json("GET", "/api/models")
        print("\nActive model:", models.get("active"))

        export_payload = {"session_id": SESSION_ID}
        export_json = request_json("POST", "/api/export/json", export_payload)
        print("Exported turns:", len(export_json.get("turns", [])))

        return 0
    except RuntimeError as exc:
        print(f"Error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
```

</details>

<details>
<summary><strong>Go</strong></summary>

Save as `archimind_api_samples.go`, then run `go run archimind_api_samples.go`.

```go
// archimind_api_samples.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var baseURL = strings.TrimRight(getenv("ARCHIMIND_BASE_URL", "http://localhost:8090"), "/")
var sessionID = getenv("ARCHIMIND_SESSION_ID", "readme-go-demo")

type HealthResponse struct {
	Status     string `json:"status"`
	App        string `json:"app"`
	AppVersion string `json:"app_version"`
}

type ChatResponse struct {
	Answer  string `json:"answer"`
	Sources []any  `json:"sources"`
}

type ModelsResponse struct {
	Active string `json:"active"`
	Models []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"models"`
}

type ExportResponse struct {
	SessionID string `json:"session_id"`
	Turns     []any  `json:"turns"`
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func requestJSON[T any](ctx context.Context, client *http.Client, method, path string, payload any) (T, error) {
	var zero T

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return zero, fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return zero, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return zero, fmt.Errorf("archimind returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	if err := json.Unmarshal(responseBody, &zero); err != nil {
		return zero, fmt.Errorf("decode response: %w", err)
	}
	return zero, nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 60 * time.Second}

	health, err := requestJSON[HealthResponse](ctx, client, http.MethodGet, "/api/health", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Health: %s (%s %s)\n", health.Status, health.App, health.AppVersion)

	chat, err := requestJSON[ChatResponse](ctx, client, http.MethodPost, "/api/chat", map[string]string{
		"session_id": sessionID,
		"message":    "Summarise the strongest retrieved evidence in three bullets.",
		"mode":       "normal",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Chat request failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nAnswer:\n%s\n\nSources returned: %d\n", chat.Answer, len(chat.Sources))

	models, err := requestJSON[ModelsResponse](ctx, client, http.MethodGet, "/api/models", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Model lookup failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nActive model: %s\n", models.Active)

	exportResponse, err := requestJSON[ExportResponse](ctx, client, http.MethodPost, "/api/export/json", map[string]string{
		"session_id": sessionID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Exported turns: %d\n", len(exportResponse.Turns))
}
```

</details>

<details>
<summary><strong>Node</strong></summary>

Save as `archimind_api_samples.mjs`, then run `node archimind_api_samples.mjs` with Node.js 20+.

```javascript
// archimind_api_samples.mjs
const baseUrl = (process.env.ARCHIMIND_BASE_URL ?? "http://localhost:8090").replace(/\/$/, "");
const sessionId = process.env.ARCHIMIND_SESSION_ID ?? "readme-node-demo";

async function requestJson(method, path, payload = undefined) {
  const response = await fetch(`${baseUrl}${path}`, {
    method,
    headers: {
      Accept: "application/json",
      ...(payload === undefined ? {} : { "Content-Type": "application/json" }),
    },
    body: payload === undefined ? undefined : JSON.stringify(payload),
  });

  const text = await response.text();
  if (!response.ok) {
    throw new Error(`ArchiMind returned HTTP ${response.status}: ${text}`);
  }

  try {
    return JSON.parse(text);
  } catch (error) {
    throw new Error(`Could not parse JSON response: ${error.message}`);
  }
}

async function main() {
  const health = await requestJson("GET", "/api/health");
  console.log(`Health: ${health.status} (${health.app} ${health.app_version})`);

  const chat = await requestJson("POST", "/api/chat", {
    session_id: sessionId,
    message: "Summarise the strongest retrieved evidence in three bullets.",
    mode: "normal",
  });
  console.log("\nAnswer:\n", chat.answer ?? "");
  console.log("\nSources returned:", Array.isArray(chat.sources) ? chat.sources.length : 0);

  const models = await requestJson("GET", "/api/models");
  console.log("\nActive model:", models.active);

  const exported = await requestJson("POST", "/api/export/json", { session_id: sessionId });
  console.log("Exported turns:", Array.isArray(exported.turns) ? exported.turns.length : 0);
}

main().catch((error) => {
  console.error(`Error: ${error.message}`);
  process.exitCode = 1;
});
```

</details>

Additional endpoint examples:

```bash
# Health check
curl -s http://localhost:8090/api/health

# Compare two collections
curl -s http://localhost:8090/api/compare \
  -H 'Content-Type: application/json' \
  -d '{"session_id":"demo","message":"Where do these collections agree and disagree?","left_collection":"meta_reflections","right_collection":"mb_chunks","mode":"synthesis"}'

# Extract a framework from retrieved evidence
curl -s http://localhost:8090/api/framework \
  -H 'Content-Type: application/json' \
  -d '{"session_id":"demo","message":"Extract a practical decision framework from this topic."}'

# Start a background report
curl -s http://localhost:8090/api/report \
  -H 'Content-Type: application/json' \
  -d '{"topic":"retrieval quality risks"}'

# Switch the active chat model for future requests
curl -s http://localhost:8090/api/model \
  -H 'Content-Type: application/json' \
  -d '{"model":"deepseek/deepseek-r1"}'
```

## Development commands

```bash
gofmt -w .
go test ./...
go build ./...
go mod tidy
```

CI runs in `.github/workflows/tests.yml` on push/PR.
