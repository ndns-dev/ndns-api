# Go App Info
APP_NAME := ndns-go
MAIN_FILE := ./pkg/main.go
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
	go run $(MAIN_FILE)

run-prod:
	./ndns-go

# Docker 빌드 (go build 포함)
docker-build:
	docker buildx build \
		--platform $(PLATFORM) \
		--build-arg BUILD_VERSION=$(VERSION) \
		-t $(APP_NAME) . \
		--load

docker-run:
	docker run --platform $(PLATFORM) --env-file .env -p $(PORT):$(PORT) $(APP_NAME)

# Docker 내부 테서랙트 확인
docker-test-tesseract:
	docker run --rm $(APP_NAME) bash -c "which tesseract && tesseract --version && tesseract --list-langs"

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
