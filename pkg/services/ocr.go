package service

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/otiai10/gosseract/v2"
	"github.com/sh5080/ndns-go/pkg/configs"
)

// NewOCRService는 새 OCR 서비스를 생성합니다
func NewOCRService() OCRService {
	return &Service{
		config: configs.GetConfig(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExtractTextFromImage는 이미지 URL에서 텍스트를 추출합니다
func (s *Service) ExtractTextFromImage(imageURL string) (string, error) {
	// Tesseract 설치 확인
	if !tesseractInstalled() {
		return "", fmt.Errorf("Tesseract OCR이 설치되어 있지 않습니다")
	}

	// 이미지 다운로드
	tempDir, err := os.MkdirTemp(s.config.OCR.TempDir, "ocr")
	if err != nil {
		return "", fmt.Errorf("임시 디렉터리 생성 실패: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 이미지 파일 경로 설정
	imagePath := filepath.Join(tempDir, "image.jpg")

	// 이미지 다운로드
	if err := s.downloadImage(imageURL, imagePath); err != nil {
		return "", fmt.Errorf("이미지 다운로드 실패: %v", err)
	}

	// Tesseract 클라이언트 생성
	client := gosseract.NewClient()
	defer client.Close()

	// 한국어 언어 팩 설정
	if err := client.SetLanguage("kor"); err != nil {
		return "", fmt.Errorf("언어 설정 실패: %v", err)
	}

	// 이미지 설정
	if err := client.SetImage(imagePath); err != nil {
		return "", fmt.Errorf("이미지 설정 실패: %v", err)
	}

	// OCR 수행
	text, err := client.Text()
	if err != nil {
		return "", fmt.Errorf("텍스트 추출 실패: %v", err)
	}

	// 텍스트 정리
	text = strings.TrimSpace(text)

	return text, nil
}

// downloadImage는 URL에서 이미지를 다운로드합니다
func (s *Service) downloadImage(imageURL, destination string) error {
	// 이미지 URL이 올바른지 확인
	if !strings.HasPrefix(imageURL, "http") {
		return fmt.Errorf("유효하지 않은 이미지 URL: %s", imageURL)
	}

	// HTTP 요청 생성
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return fmt.Errorf("HTTP 요청 생성 실패: %v", err)
	}

	// 헤더 설정
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// 요청 실행
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 상태 확인
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("이미지 다운로드 실패: HTTP %d", resp.StatusCode)
	}

	// 저장할 파일 생성
	out, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("파일 생성 실패: %v", err)
	}
	defer out.Close()

	// 이미지 데이터 저장
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("이미지 저장 실패: %v", err)
	}

	return nil
}

// tesseractInstalled는 Tesseract가 설치되어 있는지 확인합니다
func tesseractInstalled() bool {
	cmd := exec.Command("tesseract", "--version")
	err := cmd.Run()
	return err == nil
}
