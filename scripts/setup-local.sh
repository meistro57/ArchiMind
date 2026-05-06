#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_EXAMPLE="$ROOT_DIR/.env.example"
ENV_FILE="$ROOT_DIR/.env"

log() {
  printf '[setup] %s\n' "$1"
}

require_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    log "Docker is required but was not found in PATH."
    exit 1
  fi

  if ! docker info >/dev/null 2>&1; then
    log "Docker daemon is not running. Start Docker and re-run this script."
    exit 1
  fi
}

ensure_container() {
  local name="$1"
  local image="$2"
  shift 2

  if docker container inspect "$name" >/dev/null 2>&1; then
    if [[ "$(docker container inspect -f '{{.State.Running}}' "$name")" == "true" ]]; then
      log "Container '$name' already exists and is running."
    else
      log "Container '$name' already exists."
    fi
    return
  fi

  log "Creating container '$name' from image '$image'."
  docker run -d --name "$name" "$@" "$image" >/dev/null
}

ensure_env_file() {
  if [[ -f "$ENV_FILE" ]]; then
    log ".env already exists."
    return
  fi

  if [[ ! -f "$ENV_EXAMPLE" ]]; then
    log "Missing .env.example at $ENV_EXAMPLE"
    exit 1
  fi

  cp "$ENV_EXAMPLE" "$ENV_FILE"
  log "Created .env from .env.example."
}

env_value() {
  local key="$1"
  grep "^${key}=" "$ENV_FILE" | head -n1 | cut -d'=' -f2- || true
}

upsert_env_value() {
  local key="$1"
  local value="$2"
  local escaped

  escaped="$(printf '%s' "$value" | sed 's/[\\/&]/\\&/g')"

  if grep -q "^${key}=" "$ENV_FILE"; then
    sed -i.bak "s/^${key}=.*/${key}=${escaped}/" "$ENV_FILE"
    rm -f "$ENV_FILE.bak"
  else
    printf '\n%s=%s\n' "$key" "$value" >> "$ENV_FILE"
  fi
}

upsert_openrouter_key() {
  local current_key
  current_key="$(env_value OPENROUTER_API_KEY)"

  if [[ -n "$current_key" && "$current_key" != "sk-or...." ]]; then
    log "OPENROUTER_API_KEY already set in .env."
    return
  fi

  printf 'Enter OPENROUTER_API_KEY: '
  read -r -s api_key
  printf '\n'

  if [[ -z "${api_key// }" ]]; then
    log "OPENROUTER_API_KEY cannot be empty."
    exit 1
  fi

  upsert_env_value OPENROUTER_API_KEY "$api_key"
  log "OPENROUTER_API_KEY updated in .env."
}

configure_openrouter_model() {
  local current_model
  local selected_model
  local choice

  current_model="$(env_value OPENROUTER_MODEL)"

  printf '\nSelect default OpenRouter chat model for OPENROUTER_MODEL\n'
  printf '  1) deepseek/deepseek-r1\n'
  printf '  2) openai/gpt-4.1-mini\n'
  printf '  3) anthropic/claude-3.5-sonnet\n'
  printf '  4) google/gemini-2.0-flash-001\n'
  printf '  5) Enter custom model ID\n'
  if [[ -n "$current_model" ]]; then
    printf '  6) Keep current (%s)\n' "$current_model"
  fi

  printf 'Choice [%s]: ' "$( [[ -n "$current_model" ]] && printf '6' || printf '1' )"
  read -r choice

  if [[ -z "$choice" ]]; then
    if [[ -n "$current_model" ]]; then
      choice="6"
    else
      choice="1"
    fi
  fi

  case "$choice" in
    1) selected_model="deepseek/deepseek-r1" ;;
    2) selected_model="openai/gpt-4.1-mini" ;;
    3) selected_model="anthropic/claude-3.5-sonnet" ;;
    4) selected_model="google/gemini-2.0-flash-001" ;;
    5)
      printf 'Enter custom OpenRouter model ID: '
      read -r selected_model
      if [[ -z "${selected_model// }" ]]; then
        log "Model ID cannot be empty."
        exit 1
      fi
      ;;
    6)
      if [[ -z "$current_model" ]]; then
        log "No current OPENROUTER_MODEL found in .env."
        exit 1
      fi
      selected_model="$current_model"
      ;;
    *)
      log "Invalid model selection."
      exit 1
      ;;
  esac

  upsert_env_value OPENROUTER_MODEL "$selected_model"
  log "OPENROUTER_MODEL set to '$selected_model'."
}

main() {
  require_docker

  ensure_container "archimind-qdrant" "qdrant/qdrant:latest" \
    --restart unless-stopped \
    -p 6333:6333 \
    -v archimind_qdrant_storage:/qdrant/storage

  ensure_container "archimind-redis" "redis:7-alpine" \
    --restart unless-stopped \
    -p 6379:6379 \
    -v archimind_redis_data:/data

  ensure_env_file
  upsert_openrouter_key
  configure_openrouter_model

  log "Setup complete."
  log "Qdrant: http://localhost:6333"
  log "Redis: localhost:6379"
  log "Next step: go run ."
}

main "$@"
