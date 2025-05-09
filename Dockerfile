# 디버그용 스테이지 (go build 문제 조사용)
FROM golang:1.24.2 AS builder

WORKDIR /app
COPY . .

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# 버전 주입을 위한 ARG 선언
ARG BUILD_VERSION=dev

RUN go mod tidy
RUN go version
RUN go env

# 1차 대안: 실패하더라도 계속 진행 (최종 이미지는 prebuilt 바이너리 사용)
RUN go build -ldflags "-X 'github.com/sh5080/ndns-go/pkg/controller.Version=${BUILD_VERSION}'" -o ndns-go ./pkg/main.go || echo "빌드 실패, 로컬 바이너리 사용 예정"

# 런타임 스테이지
FROM ubuntu:20.04

# 기본 환경 설정
ENV DEBIAN_FRONTEND=noninteractive

# 필수 패키지 설치 (Tesseract OCR, ImageMagick, 기타 도구)
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tesseract-ocr \
    tesseract-ocr-kor \
    imagemagick \
    file \
    curl \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

ENV PATH="/usr/bin:${PATH}"
ENV TESSDATA_PREFIX="/usr/share/tesseract-ocr/4.00/tessdata"

WORKDIR /app

# 로컬에서 미리 빌드한 바이너리 복사 (CI 환경에서 먼저 빌드해야 함)
# 이 파일이 없으면 Docker 빌드가 실패합니다
# COPY ./ndns-go .
#builder 스테이지에서 빌드에 성공한 경우 사용
COPY --from=builder /app/ndns-go . 

EXPOSE 8085

CMD ["./ndns-go"]
