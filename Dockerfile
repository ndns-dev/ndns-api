# 빌더 스테이지
FROM golang:1.24.2 AS builder

WORKDIR /app
COPY . .

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# 버전 주입을 위한 ARG 선언
ARG BUILD_VERSION=dev

RUN go mod tidy

# Version 심볼 주입 (패키지 경로 정확히 명시)
RUN go build -ldflags "-X 'github.com/sh5080/ndns-go/pkg/controllers.Version=${BUILD_VERSION}'" -o ndns-go ./pkg/main.go

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
COPY --from=builder /app/ndns-go .
COPY .env .

EXPOSE 8085

CMD ["./ndns-go"]
