#!/bin/bash

set -e

IMAGE=sh5080/ndns-go:latest
OLD_CONTAINER_NAME=ndns-go
NEW_CONTAINER_NAME=ndns-go-next
INTERNAL_PORT=8085
ENV_FILE_PATH="/home/ubuntu/ndns-go/.env"
NGINX_CONF_PATH="/etc/nginx/conf.d/ndns-go.conf"
NGINX_TEMPLATE="/home/ubuntu/deploy/nginx_template.conf"


# 시스템에서 포트 사용 여부 확인
is_port_in_use() {
  ss -ltn | awk '{print $4}' | grep -q ":$1$"
}

# 포트 선택 (호스트에서 컨테이너로 포워딩할 포트)
echo "🔍 Finding available port..."
for PORT in {8087..8099}; do
  if ! is_port_in_use "$PORT"; then
    NEXT_PORT=$PORT
    break
  fi
done

if [ -z "$NEXT_PORT" ]; then
  echo "❌ No available port found in range 8087–8099"
  exit 1
fi

echo "✅ Using internal forwarding port $NEXT_PORT (public stays on 8086)"

echo "📦 Pulling latest Docker image..."
docker pull $IMAGE

echo "🧹 Cleaning up existing $NEW_CONTAINER_NAME container if exists..."
docker rm -f $NEW_CONTAINER_NAME 2>/dev/null || true

echo "🚀 Starting new container $NEW_CONTAINER_NAME..."
docker run -d \
  --env-file "$ENV_FILE_PATH" \
  -p 127.0.0.1:$NEXT_PORT:$INTERNAL_PORT \
  --name $NEW_CONTAINER_NAME \
  $IMAGE

echo "⏳ Waiting for health check..."
sleep 3

if ! curl -s http://127.0.0.1:$NEXT_PORT/health | grep -q "ok"; then
  echo "❌ Health check failed. Stopping deployment."
  docker rm -f $NEW_CONTAINER_NAME
  exit 1
fi

echo "✅ Health check passed. Updating NGINX config (8086 → $NEXT_PORT)..."
sed "s/{{PORT}}/$NEXT_PORT/g" $NGINX_TEMPLATE | sudo tee $NGINX_CONF_PATH > /dev/null

echo "🔁 Reloading NGINX..."
sudo nginx -t && sudo systemctl reload nginx

echo "🧹 Cleaning up old container..."
docker rm -f $OLD_CONTAINER_NAME || true
docker rename $NEW_CONTAINER_NAME $OLD_CONTAINER_NAME

echo "✅ Deployment completed successfully."
