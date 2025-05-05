package detector

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	repository "github.com/sh5080/ndns-go/pkg/repositories"
)

// OCRImpl는 OCR 서비스 구현체입니다
type OCRImpl struct {
	_interface.Service
	ocrRepo _interface.OCRRepository
}

// NewOCRService는 새 OCR 서비스를 생성합니다
func NewOCRService() _interface.OCRService {
	return &OCRImpl{
		Service: _interface.Service{
			Client: &http.Client{
				Timeout: time.Second * 30,
			},
			Config: configs.GetConfig(),
		},
		ocrRepo: repository.NewOCRRepository(),
	}
}

// ExtractTextFromImage는 이미지 URL에서 텍스트를 추출합니다
func (o *OCRImpl) ExtractTextFromImage(imageURL string) (string, error) {
	if imageURL == "" {
		return "", fmt.Errorf("이미지 URL이 비어 있습니다")
	}

	// 캐시 확인
	if o.ocrRepo != nil {
		cache, err := o.ocrRepo.GetOCRCache(imageURL)
		if err == nil && cache != nil && cache.TextDetected != "" {
			return cache.TextDetected, nil
		}
	}

	// Tesseract 설치 확인
	if !o.isTesseractInstalled() {
		return "", fmt.Errorf("tesseract OCR이 설치되지 않았습니다")
	}

	// 이미지 다운로드
	tempFile, err := o.downloadImage(imageURL)
	if err != nil {
		return "", fmt.Errorf("이미지 다운로드 실패: %v", err)
	}
	defer os.Remove(tempFile) // 임시 파일 정리

	// OCR 실행
	textDetected, err := o.runOCR(tempFile)
	if err != nil {
		return "", fmt.Errorf("OCR 실행 실패: %v", err)
	}
	fmt.Printf("textDetected: %v\n", textDetected)
	// OCR 결과 캐싱
	if o.ocrRepo != nil && textDetected != "" {
		// 비동기 저장 (결과에 영향 없음)
		go func() {
			_ = o.ocrRepo.SaveOCRCache(imageURL, textDetected, "image")
		}()
	}

	return textDetected, nil
}

// downloadImage는 이미지 URL에서 이미지를 다운로드합니다
func (o *OCRImpl) downloadImage(imageURL string) (string, error) {
	// URL 정규화
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		imageURL = "https://" + imageURL
	}

	// HTTP 요청 생성
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("요청 생성 실패: %v", err)
	}

	// 요청 헤더 추가 (브라우저 에뮬레이션)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Add("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Add("Accept-Language", "ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7")

	// 요청 실행
	resp, err := o.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("요청 실행 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 상태 확인
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP 오류 (%d)", resp.StatusCode)
	}

	// 임시 파일 생성
	tempDir := os.TempDir()
	tempFileName := uuid.New().String() + ".jpg"
	tempFilePath := filepath.Join(tempDir, tempFileName)

	// 이미지 파일 저장
	file, err := os.Create(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("임시 파일 생성 실패: %v", err)
	}
	defer file.Close()

	// 이미지 데이터 복사
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("이미지 저장 실패: %v", err)
	}

	return tempFilePath, nil
}

// runOCR은 이미지 파일에서 텍스트를 추출합니다
func (o *OCRImpl) runOCR(imagePath string) (string, error) {
	// OCR 디버깅용
	fmt.Printf("OCR 실행 - 이미지 경로: %s\n", imagePath)

	// 첫 번째 시도: 파이썬 코드와 동일한 설정으로 실행
	// lang="kor", config="--psm 6 --oem 3 -c preserve_interword_spaces=1"
	cmd := exec.Command("tesseract",
		imagePath,
		"stdout",
		"-l", "kor",
		"--psm", "6",
		"--oem", "3",
		"-c", "preserve_interword_spaces=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("OCR 실행 실패: %v\n", err)
		// 에러가 있어도, 출력 내용은 확인
	}

	// OCR 결과 정리
	textDetected := strings.TrimSpace(string(output))

	// 결과가 없거나 디버그 메시지만 있는 경우
	if textDetected == "" || strings.Contains(textDetected, "Estimating") {
		fmt.Printf("OCR 결과 미흡, 두 번째 시도\n")

		// 두 번째 시도: 간단한 설정으로 재시도
		cmdAlt := exec.Command("tesseract",
			imagePath,
			"stdout",
			"-l", "kor")

		outputAlt, err := cmdAlt.CombinedOutput()
		if err != nil {
			fmt.Printf("두 번째 OCR 시도 실패: %v\n", err)
			// 첫 번째 결과라도 반환
		} else {
			altText := strings.TrimSpace(string(outputAlt))
			if altText != "" && !strings.Contains(altText, "Estimating") {
				textDetected = altText
			}
		}
	}

	// 결과 확인 및 길이 출력
	if textDetected == "" {
		fmt.Printf("OCR 결과: 추출된 텍스트 없음\n")
	} else {
		// 출력 준비
		previewLen := 50
		if len(textDetected) < previewLen {
			previewLen = len(textDetected)
		}

		preview := textDetected[:previewLen]
		suffix := ""
		if len(textDetected) > previewLen {
			suffix = "..."
		}

		fmt.Printf("OCR 결과 (%d 바이트): %s%s\n", len(textDetected), preview, suffix)
	}

	// 디버그 메시지만 있고 실제 텍스트가 없는 경우
	if strings.Contains(textDetected, "Estimating") && len(textDetected) < 100 {
		return "", nil
	}

	return textDetected, nil
}

// min은 두 정수 중 작은 값을 반환합니다
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isTesseractInstalled는 Tesseract OCR이 설치되어 있는지 확인합니다
func (o *OCRImpl) isTesseractInstalled() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}
