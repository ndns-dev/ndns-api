package detector

import (
	"fmt"
	"time"

	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	"github.com/sh5080/ndns-go/pkg/services/internal/queue"
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

// OcrService는 Ocr 처리를 관리하는 서비스입니다
type OcrService struct {
	queueService    _interface.QueueService
	analyzerService _interface.AnalyzerService
	config          *configs.EnvConfig
}

// ProcessOcrAndRequestNext는 Ocr 결과를 처리하고 필요한 경우 다음 Ocr을 요청합니다
func (s *OcrService) ProcessOcrAndRequestNext(state model.OcrQueueState, ocrResult model.OcrResult) (*_interface.OcrProcessResponse, error) {
	// OCR 텍스트 분석
	result, err := s.analyzerService.AnalyzeCycle(state, ocrResult)
	if err != nil {
		return nil, err
	}

	// 협찬이 발견되지 않은 경우 다음 Ocr 요청
	if !result.IsSponsored {
		nextPosition := model.GetNextOcrPosition(state.CurrentPosition, state.Is2025OrLater)
		if nextPosition != "" {
			// 다음 Ocr 요청
			state.CurrentPosition = nextPosition
			err := s.RequestNextOcr(state)
			if err != nil {
				return nil, fmt.Errorf("다음 Ocr 요청 실패: %v", err)
			}
		}
	}

	// 협찬이 발견된 경우 처리는 나중에 구현
	return nil, nil
}

// NewOcrService는 새로운 Ocr 처리 서비스를 생성합니다
func NewOcrService(queueService _interface.QueueService, analyzerService _interface.AnalyzerService) _interface.OcrService {
	if queueService == nil {
		queueService = queue.NewSqsService()
	}

	return &OcrService{
		queueService:    queueService,
		analyzerService: analyzerService,
		config:          configs.GetConfig(),
	}
}

// RequestNextOcr은 다음 Ocr 처리를 요청합니다
func (s *OcrService) RequestNextOcr(state model.OcrQueueState) error {
	nextMessage := &model.OcrQueueState{
		JobId:           state.JobId,
		RequestedAt:     time.Now(),
		Is2025OrLater:   state.Is2025OrLater,
		CrawlResult:     state.CrawlResult,
		CurrentPosition: state.CurrentPosition,
	}

	switch state.CurrentPosition {
	case model.OcrPositionFirstImage:
		if state.CrawlResult.FirstStickerUrl != "" {
			nextMessage.CurrentPosition = model.OcrPositionFirstSticker
		}
	case model.OcrPositionFirstSticker:
		if state.CrawlResult.SecondStickerUrl != "" && state.CrawlResult.SecondStickerUrl != state.CrawlResult.FirstStickerUrl {
			nextMessage.CurrentPosition = model.OcrPositionSecondSticker
		}
	case model.OcrPositionSecondSticker:
		if state.CrawlResult.LastImageUrl != "" && state.CrawlResult.LastImageUrl != state.CrawlResult.FirstImageUrl {
			nextMessage.CurrentPosition = model.OcrPositionLastImage
		}
	case model.OcrPositionLastImage:
		if state.CrawlResult.LastStickerUrl != "" && state.CrawlResult.LastStickerUrl != state.CrawlResult.FirstStickerUrl {
			nextMessage.CurrentPosition = model.OcrPositionLastSticker
		}
	}

	return s.queueService.SendQueue(*nextMessage)
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
