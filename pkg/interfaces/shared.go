package _interface

import (
	"net/http"

	"github.com/sh5080/ndns-go/pkg/configs"
)

type Service struct {
	Config *configs.EnvConfig
	Client *http.Client
}

// ServiceContainer는 모든 서비스 인스턴스를 보관합니다
type ServiceContainer struct {
	OCRService    OCRService
	SearchService SearchService
	PostService   PostService
	OCRRepository OCRRepository
}
