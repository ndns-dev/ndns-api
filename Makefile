# Go App Info
APP_NAME := ndns-go
MAIN_FILE := ./cmd/server/main.go
VERSION := $(shell git rev-parse --short HEAD)

# Docker Info
DOCKER_IMAGE := $(APP_NAME)
PORT := 8086
PLATFORM := linux/amd64

# 로컬 Go 실행용
install:
	go mod tidy

build:
	go build -ldflags "-X 'github.com/sh5080/ndns-go/pkg/controller.Version=$(VERSION)'" -o $(APP_NAME) $(MAIN_FILE)

run:
	APP_ENV=dev go run $(MAIN_FILE)

run-prod:
	./ndns-go

# 크로스 컴파일 (리눅스용 바이너리 생성)
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X 'github.com/sh5080/ndns-go/pkg/controller.Version=$(VERSION)'" -o $(APP_NAME) $(MAIN_FILE)

# Docker 빌드 (go build 포함)
docker-build: build-linux
	docker buildx build \
		--platform $(PLATFORM) \
		--build-arg BUILD_VERSION=$(VERSION) \
		-t $(APP_NAME) . \
		--load

# Docker 빌드 (빌드 과정 스킵, 로컬 바이너리 사용)
docker-build-skip-go:
	@if [ ! -f $(APP_NAME) ]; then \
		echo "로컬 바이너리가 없습니다. 빌드 중..."; \
		$(MAKE) build-linux; \
	fi
	docker buildx build \
		--platform $(PLATFORM) \
		-t $(APP_NAME) . \
		--load

docker-run:
	docker run --platform $(PLATFORM) --env-file .env -p $(PORT):$(PORT) $(APP_NAME)

docker-push-cloudrun:
	docker build -t gcr.io/axial-analyzer-460001-p3/ndns-go-cloudrun -f Dockerfile.cloudrun .
	gcloud builds submit --config=cloudbuild.yaml .

# Docker 쉘 진입
docker-shell:
	docker run --rm -it $(APP_NAME) bash

# Docker 배포 준비용 빌드 (tag 지정 예시)
docker-push:
	docker tag $(APP_NAME) sh5080/$(APP_NAME):latest
	docker push sh5080/$(APP_NAME):latest

# Docker 이미지 삭제
docker-clean:
	docker rmi $(DOCKER_IMAGE)

# 로컬 바이너리 삭제
clean:
	rm -f $(APP_NAME)
