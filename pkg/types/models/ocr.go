package model

import (
	"time"

	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// OCRCache는 DynamoDB에 저장될 OCR 캐시 아이템을 나타냅니다.
type OCRCache struct {
	ImageURL     string              `json:"imageUrl"`     // 프라이머리 키
	TextDetected string              `json:"textDetected"` // OCR 결과 텍스트
	ImageType    structure.BlogImage `json:"imageType"`    // 이미지 타입
	Result       string              `json:"result"`       // OCR 결과
	CreatedAt    time.Time           `json:"createdAt"`    // 생성 시간
	ExpiresAt    time.Time           `json:"expiresAt"`    // 만료 시간
}
