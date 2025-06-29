package detector

import (
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	"github.com/sh5080/ndns-go/pkg/services/internal/common"
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

// OcrService는 OCR 서비스 구현체입니다
type OcrService struct {
	_interface.Service
	queueService _interface.QueueService
}

// NewOcrService는 새로운 OCR 서비스를 생성합니다
func NewOcrService(queueService _interface.QueueService) _interface.OcrService {
	return &OcrService{
		queueService: queueService,
	}
}

// RequestNextOcr은 다음 Ocr 처리를 요청합니다
func (s *OcrService) RequestNextOcr(state model.OcrQueueState) error {
	nextPosition := common.GetNextOcrPosition(state.CurrentPosition, state.Is2025OrLater)

	state.CurrentPosition = nextPosition
	return s.queueService.SendQueue(state)
}

// CreatePendingIndicator는 Ocr 분석 중임을 나타내는 지표를 생성합니다
func CreatePendingIndicator(jobId string) structure.SponsorIndicator {
	return structure.SponsorIndicator{
		Type:        structure.IndicatorTypePending,
		Pattern:     structure.PatternTypeNormal,
		MatchedText: "분석 중입니다. 잠시만 기다려주세요.",
		Probability: 0,
		Source: structure.SponsorSource{
			SponsorType: structure.SponsorTypeImage,
			Text:        jobId,
		},
	}
}
