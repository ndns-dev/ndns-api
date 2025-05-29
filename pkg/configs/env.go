package configs

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

// 앱 버전을 저장하는 전역 변수
var AppVersion string

type EnvConfig struct {
	Server struct {
		Port      string `env:"PORT,required"`
		WorkerURL string `env:"WORKER_URL,required"`
		AppName   string `env:"APP_NAME,required"`
		AppURL    string `env:"APP_URL,required"`
	}
	AWS struct {
		AccessKeyID      string `env:"AWS_ACCESS_KEY_ID,required"`
		SecretAccessKey  string `env:"AWS_SECRET_ACCESS_KEY,required"`
		Region           string `env:"AWS_REGION,required"`
		DynamoDBEndpoint string `env:"AWS_DYNAMODB_ENDPOINT" envDefault:""`
		Tables           struct {
			OCRCache     string `env:"AWS_DYNAMODB_TABLE_OCR_CACHE"`
			ServerStatus string `env:"AWS_DYNAMODB_TABLE_SERVER_STATUS"`
			Server       string `env:"AWS_DYNAMODB_TABLE_SERVER"`
		}
	}
	Naver struct {
		ClientID     string `env:"NAVER_CLIENT_ID,required"`
		ClientSecret string `env:"NAVER_CLIENT_SECRET,required"`
		SearchURL    string `env:"NAVER_SEARCH_URL" envDefault:"https://openapi.naver.com/v1/search/blog.json"`
	}
	OCR struct {
		TesseractPath string `env:"OCR_TESSERACT_PATH" envDefault:"/usr/local/bin/tesseract"`
		TempDir       string `env:"OCR_TEMP_DIR" envDefault:"/tmp"`
	}
}

var (
	configInstance *EnvConfig
	once           sync.Once
)

func init() {
	AppVersion = getEnvOrDefault("VERSION", "dev")
	if getEnvOrDefault("APP_ENV", "") == "dev" {
		AppVersion = "dev"
	}
	fmt.Printf("앱 버전 설정: %s\n", AppVersion)
}

// getEnvOrDefault returns environment variable value or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetConfig는 EnvConfig의 싱글톤 인스턴스를 반환합니다.
func GetConfig() *EnvConfig {
	once.Do(func() {
		// .env 파일 로드 시도
		if err := godotenv.Load(); err != nil {
			log.Printf(".env 파일 로드 실패 (무시됨): %v", err)
		}

		config := &EnvConfig{}
		if err := env.Parse(config); err != nil {
			log.Fatalf("환경 변수 로드 실패: %v", err)
		}

		configInstance = config
		log.Printf("환경 변수 로드 완료 (앱 버전: %s)\n", AppVersion)
	})
	return configInstance
}
