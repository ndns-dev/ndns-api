package model

import (
	"time"

	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// OcrPosition은 Ocr을 수행할 이미지의 위치를 나타냅니다
type OcrPosition string

const (
	OcrPositionStart         OcrPosition = "Start"
	OcrPositionFirstImage    OcrPosition = "FirstImageUrl"
	OcrPositionFirstSticker  OcrPosition = "FirstStickerUrl"
	OcrPositionSecondSticker OcrPosition = "SecondStickerUrl"
	OcrPositionLastImage     OcrPosition = "LastImageUrl"
	OcrPositionLastSticker   OcrPosition = "LastStickerUrl"
)

// OcrResult는 DynamoDB에 저장될 Ocr 결과 아이템을 나타냅니다.
type OcrResult struct {
	ImageUrl    string      `json:"imageUrl" dynamodbav:"imageUrl"`       // 프라이머리 키
	JobId       string      `json:"jobId" dynamodbav:"jobId"`             // State 키
	Position    OcrPosition `json:"position" dynamodbav:"position"`       // Ocr 위치
	OcrText     string      `json:"ocrText" dynamodbav:"ocrText"`         // Ocr 결과 텍스트
	ProcessedAt time.Time   `json:"processedAt" dynamodbav:"processedAt"` // 처리 시간
	Error       string      `json:"error" dynamodbav:"error"`             // 오류 메시지
}

// OcrQueueState는 Ocr 처리 상태를 관리합니다
type OcrQueueState struct {
	JobId           string                 `json:"jobId" dynamodbav:"jobId"`                     // 프라이머리 키
	CrawlResult     *structure.CrawlResult `json:"crawlResult" dynamodbav:"crawlResult"`         // 크롤링 결과
	CurrentPosition OcrPosition            `json:"currentPosition" dynamodbav:"currentPosition"` // 현재 Ocr 위치
	Is2025OrLater   bool                   `json:"is2025OrLater" dynamodbav:"is2025OrLater"`     // 2025년 이후 여부
	RequestedAt     time.Time              `json:"requestedAt" dynamodbav:"requestedAt"`         // 요청 시간
}
