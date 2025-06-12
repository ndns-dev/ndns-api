# 빌더 스테이지
FROM golang:1.24.2 AS builder

WORKDIR /app

# Go 모듈 파일 복사 및 의존성 다운로드
COPY go.mod go.sum ./
RUN go mod download

# 소스 코드 복사 및 빌드
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o ndns-go ./cmd/server/main.go

# 런타임 스테이지
FROM sh5080/tesseract:latest

WORKDIR /app

# builder 스테이지에서 빌드된 ndns-go 바이너리 복사
COPY --from=builder /app/ndns-go .

# 실행 권한 설정
RUN chmod +x /app/ndns-go

EXPOSE 8085

CMD ["./ndns-go"]
