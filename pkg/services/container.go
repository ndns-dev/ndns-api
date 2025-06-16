package service

import (
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	repository "github.com/sh5080/ndns-go/pkg/repositories"
	"github.com/sh5080/ndns-go/pkg/services/api"
	"github.com/sh5080/ndns-go/pkg/services/internal/detector"
)

// NewServiceContainer는 새로운 서비스 컨테이너를 생성합니다
func NewServiceContainer() *_interface.ServiceContainer {
	searchService := api.NewSearchService()
	ocrService := detector.NewOCRService()
	postService := detector.NewPostService(ocrService)
	ocrRepository := repository.NewOCRRepository()

	return &_interface.ServiceContainer{
		SearchService: searchService,
		OCRService:    ocrService,
		PostService:   postService,
		OCRRepository: ocrRepository,
	}
}
