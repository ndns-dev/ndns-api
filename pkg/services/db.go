package service

import (
	"time"

	"github.com/sh5080/ndns-go/pkg/configs"
	"github.com/sh5080/ndns-go/pkg/types/structures"
)

// dbService는 DBService 인터페이스 구현체입니다
type dbService struct {
	config *configs.EnvConfig
	cache  map[string]*structures.OCRCache // 메모리 기반 캐시로 단순화
}

// NewDBService는 새 DB 서비스를 생성합니다
func NewDBService() (DBService, error) {
	return &dbService{
		config: configs.GetConfig(),
		cache:  make(map[string]*structures.OCRCache),
	}, nil
}

// GetOCRCache는 이미지 URL에 대한 OCR 캐시를 가져옵니다
func (s *dbService) GetOCRCache(imageURL string) (*structures.OCRCache, error) {
	cache, ok := s.cache[imageURL]
	if !ok {
		return nil, nil // 캐시 없음
	}
	return cache, nil
}

// SaveOCRCache는 이미지 URL에 대한 OCR 결과를 저장합니다
func (s *dbService) SaveOCRCache(imageURL string, textDetected string, imageType string) error {
	s.cache[imageURL] = &structures.OCRCache{
		ImageURL:     imageURL,
		TextDetected: textDetected,
		ImageType:    imageType,
		DetectedAt:   time.Now(),
	}
	return nil
}
