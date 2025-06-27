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
	searchService := api.NewSearchService()
	queueService := queue.NewSqsService()
	analyzerService := analyzer.NewAnalyzerService()
	ocrService := detector.NewOcrService(queueService, analyzerService)
	postService := detector.NewPostService(ocrService)
	ocrRepository := repository.NewOcrRepository()

	return &_interface.ServiceContainer{
		SearchService:   searchService,
		OcrService:      ocrService,
		PostService:     postService,
		AnalyzerService: analyzerService,
		OcrRepository:   ocrRepository,
	}
}
