#!/bin/bash

# 사용법: ./ec2-ssh.sh <INSTANCE_NAME> <KEY_PATH> [SSH_USER] [AWS_PROFILE]

INSTANCE_NAME=$1
KEY_PATH=$2
SSH_USER=${3:-ec2-user}
AWS_PROFILE=${4:-default}

if [[ -z "$INSTANCE_NAME" || -z "$KEY_PATH" ]]; then
  echo "Usage: $0 <INSTANCE_NAME> <KEY_PATH> [SSH_USER] [AWS_PROFILE]"
  exit 1
fi

echo "🔍 Fetching public IP for instance with tag Name=$INSTANCE_NAME using profile $AWS_PROFILE..."

PUBLIC_IP=$(aws ec2 describe-instances \
  --profile "$AWS_PROFILE" \
  --region ap-northeast-2 \
  --filters "Name=tag:Name,Values=$INSTANCE_NAME" "Name=instance-state-name,Values=running" \
  --query "Reservations[*].Instances[*].PublicIpAddress" \
  --output text)


if [[ -z "$PUBLIC_IP" ]]; then
  echo "❌ Instance not found or no public IP assigned."
  exit 1
fi

echo "✅ Found IP: $PUBLIC_IP"
echo "🚀 Connecting via SSH..."
ssh -i "$KEY_PATH" "$SSH_USER@$PUBLIC_IP"
