#!/bin/bash

# 사용법: ./gcp-ssh.sh <INSTANCE_NAME> <KEY_PATH> [SSH_USER] [PROJECT_ID] [ZONE]

INSTANCE_NAME=$1
KEY_PATH=$2
SSH_USER=${3:-$USER}
PROJECT_ID=${4:-$(gcloud config get-value project)}
ZONE=${5:-us-central1-f}

if [[ -z "$INSTANCE_NAME" || -z "$KEY_PATH" ]]; then
  echo "Usage: $0 <INSTANCE_NAME> <KEY_PATH> [SSH_USER] [PROJECT_ID] [ZONE]"
  exit 1
fi

if [[ -z "$PROJECT_ID" ]]; then
  echo "❌ Project ID not set. Please provide a project ID or set it with 'gcloud config set project <PROJECT_ID>'."
  exit 1
fi

if [[ -z "$ZONE" ]]; then
  echo "❌ Zone not set. Please provide a zone or set it with 'gcloud config set compute/zone <ZONE>'."
  exit 1
fi

echo "🔍 Fetching external IP for instance $INSTANCE_NAME in project $PROJECT_ID, zone $ZONE..."

EXTERNAL_IP=$(gcloud compute instances describe "$INSTANCE_NAME" \
  --project="$PROJECT_ID" \
  --zone="$ZONE" \
  --format="get(networkInterfaces[0].accessConfigs[0].natIP)")

if [[ -z "$EXTERNAL_IP" ]]; then
  echo "❌ Instance not found or no external IP assigned."
  exit 1
fi

echo "✅ Found IP: $EXTERNAL_IP"
echo "🚀 Connecting via SSH..."
ssh -i "$KEY_PATH" "$SSH_USER@$EXTERNAL_IP" 