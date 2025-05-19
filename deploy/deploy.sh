#!/bin/bash
set -e

IMAGE=sh5080/ndns-go:latest
OLD_CONTAINER=ndns-go
NEW_CONTAINER=ndns-go-next
INTERNAL_PORT=8085
ENV_FILE_PATH="/home/ubuntu/ndns-go/.env"
NGINX_INTERNAL_CONF="/etc/nginx/conf.d/ndns-go.conf"
NGINX_INTERNAL_TEMPLATE="/home/ubuntu/deploy/nginx_template.conf"
NGINX_EXTERNAL_CONF="/etc/nginx/conf.d/ndns-go-external.conf"
NGINX_EXTERNAL_TEMPLATE="/home/ubuntu/deploy/nginx_http.conf.template"

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

# 내부용 Nginx 설정 템플릿에서 포트 치환 및 적용
sed "s/{{PORT}}/$NEXT_PORT/g" "$NGINX_INTERNAL_TEMPLATE" | sudo tee "$NGINX_INTERNAL_CONF" > /dev/null

# 외부용 80포트 프록시 설정 복사 (고정)
sudo cp "$NGINX_EXTERNAL_TEMPLATE" "$NGINX_EXTERNAL_CONF"

# Nginx 설정 테스트 및 리로드
sudo nginx -t
sudo nginx -s reload

# 기존 컨테이너 종료 및 삭제
docker rm -f $OLD_CONTAINER 2>/dev/null || true

# 새 컨테이너 이름 변경
docker rename $NEW_CONTAINER $OLD_CONTAINER

echo "🎉 Deployment completed."
