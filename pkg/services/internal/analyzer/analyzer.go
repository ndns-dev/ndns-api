package analyzer

import (
	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
)

// AnalyzerService는 텍스트 분석을 위한 서비스입니다
type AnalyzerService struct {
	config                   *configs.EnvConfig
	detectorService          _interface.DetectorService
	analyzedResultRepository _interface.AnalyzedResultRepository
}

// NewAnalyzerService는 새로운 AnalyzerService를 생성합니다
func NewAnalyzerService(detectorService _interface.DetectorService, analyzedResultRepository _interface.AnalyzedResultRepository) _interface.AnalyzerService {
	return &AnalyzerService{
		config:          configs.GetConfig(),
		detectorService: detectorService,
	}
}
