#!/bin/bash
set -e

IMAGE=sh5080/ndns-go:latest
OLD_CONTAINER=ndns-go
NEW_CONTAINER=ndns-go-next
INTERNAL_PORT=8085
ENV_FILE_PATH="/home/ubuntu/ndns-go/.env"
NGINX_CONF_PATH="/etc/nginx/conf.d/ndns-go.conf"
NGINX_TEMPLATE="/home/ubuntu/deploy/nginx_template.conf"

# 사용 가능한 포트 찾기 (8087-8099)
is_port_in_use() {
  ss -ltn | awk '{print $4}' | grep -q ":$1$"
}

echo "🔍 Finding available port..."
for PORT in {8087..8099}; do
  if ! is_port_in_use "$PORT"; then
    NEXT_PORT=$PORT
    break
  fi
done

if [ -z "$NEXT_PORT" ]; then
  echo "❌ No available port found"
  exit 1
fi

echo "✅ Using port $NEXT_PORT for new container"

docker pull $IMAGE

docker rm -f $NEW_CONTAINER 2>/dev/null || true

docker run -d \
  --env-file "$ENV_FILE_PATH" \
  -p 127.0.0.1:$NEXT_PORT:$INTERNAL_PORT \
  --name $NEW_CONTAINER \
  $IMAGE

# Nginx 설정 템플릿에서 upstream 포트만 교체
sed "s/{{PORT}}/$NEXT_PORT/g" $NGINX_TEMPLATE > $NGINX_CONF_PATH

# Nginx reload
nginx -s reload

# 기존 컨테이너 종료 및 삭제
docker rm -f $OLD_CONTAINER 2>/dev/null || true

# 컨테이너 이름 변경
docker rename $NEW_CONTAINER $OLD_CONTAINER
