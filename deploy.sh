#!/bin/bash
# Deploy to Google Cloud Run
# Usage: ./deploy.sh

set -e

# ─── Configure these ────────────────────────────────────────────────────────
PROJECT_ID="${GCP_PROJECT_ID:-your-gcp-project-id}"
SERVICE_NAME="buskatotal-backend"
REGION="southamerica-east1"   # São Paulo
IMAGE="gcr.io/$PROJECT_ID/$SERVICE_NAME"
# ────────────────────────────────────────────────────────────────────────────

echo ">> Building and pushing image..."
gcloud builds submit --tag "$IMAGE" .

echo ">> Deploying to Cloud Run..."
gcloud run deploy "$SERVICE_NAME" \
  --image "$IMAGE" \
  --platform managed \
  --region "$REGION" \
  --allow-unauthenticated \
  --port 8080 \
  --set-env-vars "AUTH_MODE=jwt" \
  --set-env-vars "FIREBASE_PROJECT_ID=$PROJECT_ID" \
  --update-secrets "AUTH_JWT_SECRET=AUTH_JWT_SECRET:latest" \
  --update-secrets "PICPAY_TOKEN=PICPAY_TOKEN:latest" \
  --update-secrets "INFOCAR_ID_KEY=INFOCAR_ID_KEY:latest" \
  --update-secrets "INFOCAR_USER=INFOCAR_USER:latest" \
  --update-secrets "INFOCAR_PASSWORD=INFOCAR_PASSWORD:latest"

echo ""
echo ">> Getting service URL..."
URL=$(gcloud run services describe "$SERVICE_NAME" \
  --platform managed \
  --region "$REGION" \
  --format "value(status.url)")

echo ">> Deployed: $URL"
echo ">> Updating APP_BASE_URL..."
gcloud run services update "$SERVICE_NAME" \
  --platform managed \
  --region "$REGION" \
  --update-env-vars "APP_BASE_URL=$URL"

echo ""
echo "Done! Service is live at: $URL"
