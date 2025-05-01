package configs

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type EnvConfig struct {
	Server struct {
		Port string
	}
	AWS struct {
		AccessKeyID      string
		SecretAccessKey  string
		Region           string
		DynamoDBEndpoint string
		Tables           struct {
			OCRCache string
		}
	}
	Naver struct {
		ClientID     string
		ClientSecret string
		SearchURL    string
	}
	OCR struct {
		TesseractPath string
		TempDir       string
	}
	Weight struct {
		ExactSponsorKeywords float64
		SponsorKeywords      float64
		LowSponsorKeywords   float64
	}
}

func NewEnvConfig() (*EnvConfig, error) {
	viper.SetConfigFile(".env")
	err := viper.ReadInConfig()
	if err != nil {
		// .env 파일이 없어도 괜찮다면 이 오류 처리는 넘어가도 됩니다.
		// fmt.Println("Error reading .env file")
	}

	viper.AutomaticEnv()

	config := &EnvConfig{}

	// 필수 필드 검사 및 로드
	requiredFields := map[string]string{
		"port":                     "서버 포트",
		"aws_access_key_id":        "AWS 접근 키 ID",
		"aws_secret_access_key":    "AWS 비밀 접근 키",
		"aws_region":               "AWS 리전",
		"dynamodb_endpoint":        "AWS DynamoDB 엔드포인트",
		"dynamodb_table_ocr_cache": "AWS DynamoDB OCR 캐시 테이블",
		"naver_client_id":          "네이버 클라이언트 ID",
		"naver_client_secret":      "네이버 클라이언트 비밀 키",
	}

	for key, fieldName := range requiredFields {
		if !viper.IsSet(key) {
			return nil, fmt.Errorf(".env 파일 또는 환경 변수에 %s(%s)이 정의되어 있지 않습니다", fieldName, key)
		}
	}

	err = viper.Unmarshal(config)
	if err != nil {
		return nil, fmt.Errorf("설정 언마샬링 실패: %w", err)
	}

	// 기본값 설정 (Unmarshal 이후)
	if config.Naver.SearchURL == "" {
		viper.SetDefault("naver.search_url", "https://openapi.naver.com/v1/search/blog.json")
		config.Naver.SearchURL = viper.GetString("naver.search_url")
	}
	if config.OCR.TesseractPath == "" {
		viper.SetDefault("ocr.tesseract_path", "/usr/local/bin/tesseract")
		config.OCR.TesseractPath = viper.GetString("ocr.tesseract_path")
	}
	if config.OCR.TempDir == "" {
		viper.SetDefault("ocr.temp_dir", "/tmp")
		config.OCR.TempDir = viper.GetString("ocr.temp_dir")
	}
	viper.SetDefault("weight.exact_sponsor_keywords", 0.9)
	config.Weight.ExactSponsorKeywords = viper.GetFloat64("weight.exact_sponsor_keywords")
	viper.SetDefault("weight.sponsor_keywords", 0.7)
	config.Weight.SponsorKeywords = viper.GetFloat64("weight.sponsor_keywords")
	viper.SetDefault("weight.low_sponsor_keywords", 0.5)
	config.Weight.LowSponsorKeywords = viper.GetFloat64("weight.low_sponsor_keywords")

	return config, nil
}

func GetConfig() *EnvConfig {
	config, err := NewEnvConfig()
	if err != nil {
		log.Fatalf("설정 로드 실패: %v", err)
	}
	return config
}
