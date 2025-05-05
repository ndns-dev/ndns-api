package repository

import (
	"fmt"
	"sync"
	"time"

	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// InMemoryDBImpl는 인메모리 데이터베이스 구현체입니다
type InMemoryDBImpl struct {
	// OCR 캐시 맵 (URL -> 결과)
	ocrCache     map[string]*structure.OCRCache
	ocrCacheLock sync.RWMutex
}

// NewOCRRepository는 새 OCR 저장소를 생성합니다
func NewOCRRepository() _interface.OCRRepository {
	return &InMemoryDBImpl{
		ocrCache: make(map[string]*structure.OCRCache),
	}
}

// GetOCRCache는 이미지 URL에 대한 OCR 캐시를 가져옵니다
func (db *InMemoryDBImpl) GetOCRCache(imageURL string) (*structure.OCRCache, error) {
	if imageURL == "" {
		return nil, fmt.Errorf("이미지 URL이 비어 있습니다")
	}

	// 읽기 잠금 획득
	db.ocrCacheLock.RLock()
	defer db.ocrCacheLock.RUnlock()

	// 캐시 확인
	cache, exists := db.ocrCache[imageURL]
	if !exists {
		return nil, nil // 캐시 없음 (에러 아님)
	}

	// 캐시 만료 확인 (24시간)
	if time.Since(cache.DetectedAt) > 24*time.Hour {
		return nil, nil // 만료된 캐시
	}

	return cache, nil
}

// SaveOCRCache는 이미지 URL에 대한 OCR 결과를 저장합니다
func (db *InMemoryDBImpl) SaveOCRCache(imageURL string, textDetected string, imageType string) error {
	if imageURL == "" {
		return fmt.Errorf("이미지 URL이 비어 있습니다")
	}

	if textDetected == "" {
		return fmt.Errorf("OCR 텍스트가 비어 있습니다")
	}

	// 쓰기 잠금 획득
	db.ocrCacheLock.Lock()
	defer db.ocrCacheLock.Unlock()

	// 캐시 저장
	db.ocrCache[imageURL] = &structure.OCRCache{
		ImageURL:     imageURL,
		TextDetected: textDetected,
		ImageType:    imageType,
		DetectedAt:   time.Now(),
	}

	return nil
}
