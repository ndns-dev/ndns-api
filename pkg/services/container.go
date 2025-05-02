package service

// ServiceContainer는 모든 서비스 인스턴스를 보관합니다
type ServiceContainer struct {
	OCRService             OCRService
	CrawlerService         CrawlerService
	SearchService          SearchService
	SponsorDetectorService SponsorDetectorService
}

// NewServiceContainer는 새로운 서비스 컨테이너를 생성합니다
func NewServiceContainer() *ServiceContainer {
	ocrService := NewOCRService()
	crawlerService := NewCrawlerService()
	searchService := NewSearchService()
	sponsorService := NewSponsorDetectorService()

	return &ServiceContainer{
		OCRService:             ocrService,
		CrawlerService:         crawlerService,
		SearchService:          searchService,
		SponsorDetectorService: sponsorService,
	}
}
