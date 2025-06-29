package _interface

import (
	model "github.com/sh5080/ndns-go/pkg/types/models"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// OcrProcessResponse는 Ocr 처리 응답 구조체입니다
type OcrProcessResponse struct {
	JobId       string                       `json:"jobId"`
	IsSponsored bool                         `json:"isSponsored"`
	Probability float64                      `json:"probability"`
	Indicators  []structure.SponsorIndicator `json:"indicators"`
}

// OcrService는 Ocr 처리를 관리하는 인터페이스입니다
type OcrService interface {
	// RequestNextOcr은 다음 Ocr 처리를 요청합니다
	RequestNextOcr(state model.OcrQueueState) error
}

// OcrRepository는 Ocr 작업과 결과를 관리하는 인터페이스입니다
type OcrRepository interface {
	SaveOcrJob(jobDetail *model.OcrQueueState) error
	GetOcrJob(jobId string) (*model.OcrQueueState, error)
	SaveOcrResult(result *model.OcrResult) error
	GetOcrResult(imageUrl string) (*model.OcrResult, error)
}
