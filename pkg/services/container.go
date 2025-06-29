package service

import (
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	repository "github.com/sh5080/ndns-go/pkg/repositories"
	"github.com/sh5080/ndns-go/pkg/services/api"
	"github.com/sh5080/ndns-go/pkg/services/internal/analyzer"
	"github.com/sh5080/ndns-go/pkg/services/internal/detector"
	"github.com/sh5080/ndns-go/pkg/services/internal/queue"
)

// NewServiceContainer는 새로운 서비스 컨테이너를 생성합니다
func NewServiceContainer() *_interface.ServiceContainer {
	// 1. 기본 서비스 초기화
	queueService := queue.NewSqsService()
	ocrRepository := repository.NewOcrRepository()

	// 2. 핵심 서비스 초기화
	ocrService := detector.NewOcrService(queueService)
	analyzerService := analyzer.NewAnalyzerService(ocrService)

	// 3. 의존 서비스 초기화
	postService := detector.NewPostService(ocrService)
	searchService := api.NewSearchService(postService)

	return &_interface.ServiceContainer{
		SearchService:   searchService,
		OcrService:      ocrService,
		PostService:     postService,
		AnalyzerService: analyzerService,
		OcrRepository:   ocrRepository,
	}
}
