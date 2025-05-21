#!/bin/bash
set -e

# === 기본 설정 ===
API_IMAGE=sh5080/ndns-go:latest
OLD_API_CONTAINER=ndns-go
NEW_API_CONTAINER=ndns-go-next
INTERNAL_PORT=8085

ENV_FILE_PATH="/home/ubuntu/ndns-go/.env"
NGINX_CONF_PATH="/etc/nginx/conf.d/ndns-go.conf"
NGINX_TEMPLATE_PATH="/home/ubuntu/deploy/nginx/internal-proxy.conf.template"
COMPOSE_FILE="/home/ubuntu/deploy/docker-compose.yml"

# === 네트워크 확인 ===
echo "🌐 Checking Docker network..."
docker network ls | grep monitoring || docker network create monitoring

# === API 서버 업데이트 ===
echo "📦 Pulling latest API image..."
docker pull $API_IMAGE

echo "🔍 Finding available port..."
for PORT in {8087..8099}; do
  if ! ss -ltn | awk '{print $4}' | grep -q ":$PORT$"; then
    NEXT_PORT=$PORT
    break
  fi
done

if [ -z "$NEXT_PORT" ]; then
  echo "❌ No available port in range 8087–8099"
  exit 1
fi

echo "🧹 Removing old container $NEW_API_CONTAINER (if exists)..."
docker rm -f $NEW_API_CONTAINER 2>/dev/null || true

echo "🚀 Starting new container on port $NEXT_PORT..."
docker run -d \
  --env-file "$ENV_FILE_PATH" \
  -p 127.0.0.1:$NEXT_PORT:$INTERNAL_PORT \
  --name $NEW_API_CONTAINER \
  --network monitoring \
  $API_IMAGE

echo "⏳ Waiting for health check..."
sleep 3

if ! curl -s http://127.0.0.1:$NEXT_PORT/health | grep -q "ok"; then
  echo "❌ Health check failed!"
  docker rm -f $NEW_API_CONTAINER
  exit 1
fi

echo "✅ Health OK. Updating NGINX..."
sed "s/{{PORT}}/$NEXT_PORT/g" $NGINX_TEMPLATE_PATH | sudo tee $NGINX_CONF_PATH > /dev/null
sudo nginx -t && sudo systemctl reload nginx

echo "♻️ Swapping containers..."
docker rm -f $OLD_API_CONTAINER || true
docker rename $NEW_API_CONTAINER $OLD_API_CONTAINER

# === 모니터링 서비스 업데이트 ===
echo "📊 Updating Prometheus & Grafana with docker-compose..."

# 기존 컨테이너 제거 (이름 충돌 방지)
docker rm -f grafana prometheus 2>/dev/null || true

docker compose -f $COMPOSE_FILE pull prometheus grafana
docker compose -f $COMPOSE_FILE up -d prometheus grafana


echo "✅ All services updated."
