#!/usr/bin/env bash
#
# End-to-end demo: ingest documents → query the RAG pipeline → verify response.
# Requires the API server running at $API_URL (default: http://localhost:18080).
#
set -euo pipefail

API_URL="${API_URL:-http://localhost:18080}"
TENANT_ID="${TENANT_ID:-demo-tenant}"

echo "=== OpsPilot E2E Demo ==="
echo "API: $API_URL"
echo "Tenant: $TENANT_ID"
echo ""

# ── Step 1: Health check ──────────────────────────────────────────────
echo "── Step 1: Health check"
curl -sf "$API_URL/healthz" | head -c 100
echo ""
echo ""

# ── Step 2: Ingest documents ─────────────────────────────────────────
echo "── Step 2: Ingest documents"

curl -sf -X POST "$API_URL/api/v1/documents" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "document_id": "doc-password-reset",
    "document_version": 1,
    "source_title": "Password Reset Guide",
    "source_uri": "https://docs.example.com/password-reset",
    "content": "How to reset your password. Navigate to the Settings page in the top-right menu. Click on Security and then Reset Password. You will receive a confirmation email within 5 minutes. If you do not receive the email, check your spam folder or contact support.\n\nAccount recovery without email access. If you cannot access your registered email, contact the support team at support@example.com. Provide your account ID and a government-issued photo ID for verification. The support team will reset your credentials within 24 hours.\n\nTwo-factor authentication setup. After resetting your password, we strongly recommend enabling two-factor authentication. Go to Settings, then Security, and click Enable 2FA. Use an authenticator app like Google Authenticator or Authy for the best security."
  }' | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  Ingested: {d[\"chunks_stored\"]} chunks ({d[\"parent_chunks\"]} parents, {d[\"child_chunks\"]} children)')" 2>/dev/null || echo "  (python3 not available, check raw output)"

curl -sf -X POST "$API_URL/api/v1/documents" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "document_id": "doc-refund-policy",
    "document_version": 1,
    "source_title": "Refund Policy",
    "source_uri": "https://docs.example.com/refund-policy",
    "content": "Refund policy overview. All purchases are eligible for a full refund within 30 days of the transaction date. To request a refund, log into your account and navigate to Order History. Select the order and click Request Refund.\n\nRefund processing time. Refunds are processed within 5-7 business days after approval. The refunded amount will appear on your original payment method. If you paid with a credit card, please allow an additional billing cycle for the credit to appear.\n\nNon-refundable items. Digital downloads, gift cards, and subscription fees for the current billing period are non-refundable. Custom or personalized orders cannot be refunded once production has begun."
  }' | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  Ingested: {d[\"chunks_stored\"]} chunks ({d[\"parent_chunks\"]} parents, {d[\"child_chunks\"]} children)')" 2>/dev/null || echo "  (check raw output)"

echo ""

# ── Step 3: Create a session ─────────────────────────────────────────
echo "── Step 3: Create session"
SESSION_RESP=$(curl -sf -X POST "$API_URL/api/v1/sessions" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "'"$TENANT_ID"'", "user_id": "demo-user"}')
SESSION_ID=$(echo "$SESSION_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['session_id'])" 2>/dev/null || echo "unknown")
echo "  Session: $SESSION_ID"
echo ""

# ── Step 4: Chat query (full RAG pipeline) ───────────────────────────
echo "── Step 4: Chat query — 'How do I reset my password?'"
echo ""
curl -sf -X POST "$API_URL/api/v1/chat/stream" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "user_id": "demo-user",
    "session_id": "'"$SESSION_ID"'",
    "mode": "chat",
    "user_message": "How do I reset my password?",
    "client_request_id": "demo-req-1"
  }' 2>&1

echo ""
echo ""

# ── Step 5: Second query (different topic) ───────────────────────────
echo "── Step 5: Chat query — 'What is the refund policy?'"
echo ""
curl -sf -X POST "$API_URL/api/v1/chat/stream" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "user_id": "demo-user",
    "session_id": "'"$SESSION_ID"'",
    "mode": "chat",
    "user_message": "What is the refund policy?",
    "client_request_id": "demo-req-2"
  }' 2>&1

echo ""
echo ""
echo "=== Demo complete ==="
echo "Full RAG pipeline exercised: HyDE → Hybrid Search → Re-rank → CRAG → LitM → LLM"
