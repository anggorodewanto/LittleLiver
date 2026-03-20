#!/usr/bin/env bash
#
# deploy.sh — Deploy LittleLiver to fly.io
#
# Usage:
#   ./scripts/deploy.sh              # Deploy only
#   ./scripts/deploy.sh --secrets    # Set secrets then deploy
#   ./scripts/deploy.sh --smoke      # Deploy then run smoke test
#   ./scripts/deploy.sh --all        # Set secrets, deploy, smoke test
#
set -euo pipefail

APP_NAME="littleliver"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

# ──────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────

info()  { echo "==> $*"; }
error() { echo "ERROR: $*" >&2; exit 1; }

check_flyctl() {
    if ! command -v fly &>/dev/null; then
        error "fly CLI not found. Install from https://fly.io/docs/flyctl/install/"
    fi
}

# ──────────────────────────────────────────────
# Set secrets (interactive — prompts for missing values)
# ──────────────────────────────────────────────

set_secrets() {
    info "Setting fly.io secrets for $APP_NAME"

    local secrets=(
        "GOOGLE_CLIENT_ID"
        "GOOGLE_CLIENT_SECRET"
        "SESSION_SECRET"
        "R2_ACCOUNT_ID"
        "R2_ACCESS_KEY_ID"
        "R2_SECRET_ACCESS_KEY"
        "R2_BUCKET_NAME"
        "VAPID_PUBLIC_KEY"
        "VAPID_PRIVATE_KEY"
        "BASE_URL"
    )

    local args=()
    for secret in "${secrets[@]}"; do
        val="${!secret:-}"
        if [ -z "$val" ]; then
            read -rp "  $secret: " val
            if [ -z "$val" ]; then
                echo "  (skipping $secret — empty)"
                continue
            fi
        fi
        args+=("$secret=$val")
    done

    if [ ${#args[@]} -gt 0 ]; then
        fly secrets set "${args[@]}" --app "$APP_NAME"
        info "Secrets set successfully"
    else
        info "No secrets to set"
    fi
}

# ──────────────────────────────────────────────
# Deploy
# ──────────────────────────────────────────────

deploy() {
    info "Deploying $APP_NAME to fly.io"
    fly deploy --app "$APP_NAME"
    info "Deploy complete"
}

# ──────────────────────────────────────────────
# Smoke test
# ──────────────────────────────────────────────

smoke_test() {
    info "Running smoke tests"

    local base_url
    base_url="https://${APP_NAME}.fly.dev"

    # 1. Health endpoint
    info "Testing health endpoint..."
    local health_status
    health_status=$(curl -s -o /dev/null -w "%{http_code}" "$base_url/health")
    if [ "$health_status" = "200" ]; then
        info "Health check PASSED (HTTP 200)"
    else
        error "Health check FAILED (HTTP $health_status)"
    fi

    local health_body
    health_body=$(curl -s "$base_url/health")
    if echo "$health_body" | grep -q '"status":"ok"'; then
        info "Health response body PASSED"
    else
        error "Health response body unexpected: $health_body"
    fi

    # 2. OAuth redirect
    info "Testing OAuth redirect..."
    local oauth_status
    oauth_status=$(curl -s -o /dev/null -w "%{http_code}" "$base_url/auth/google/login")
    if [ "$oauth_status" = "302" ] || [ "$oauth_status" = "307" ]; then
        info "OAuth redirect PASSED (HTTP $oauth_status)"
    else
        echo "  OAuth redirect returned HTTP $oauth_status (expected 302/307)"
        echo "  This may be expected if GOOGLE_CLIENT_ID is not set"
    fi

    # 3. Static files (frontend)
    info "Testing static file serving..."
    local static_status
    static_status=$(curl -s -o /dev/null -w "%{http_code}" "$base_url/")
    if [ "$static_status" = "200" ]; then
        info "Static files PASSED (HTTP 200)"
    else
        echo "  Static files returned HTTP $static_status"
    fi

    # 4. HTTPS enforcement
    info "Testing HTTPS enforcement..."
    local http_status
    http_status=$(curl -s -o /dev/null -w "%{http_code}" --max-redirs 0 "http://${APP_NAME}.fly.dev/health" 2>/dev/null || true)
    if [ "$http_status" = "301" ] || [ "$http_status" = "308" ]; then
        info "HTTPS redirect PASSED (HTTP $http_status)"
    else
        echo "  HTTPS redirect returned HTTP $http_status"
    fi

    info "Smoke tests complete"
}

# ──────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────

check_flyctl

case "${1:-}" in
    --secrets)
        set_secrets
        deploy
        ;;
    --smoke)
        deploy
        smoke_test
        ;;
    --all)
        set_secrets
        deploy
        smoke_test
        ;;
    "")
        deploy
        ;;
    *)
        echo "Usage: $0 [--secrets|--smoke|--all]"
        exit 1
        ;;
esac
