package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// SaveResponseToFile은 HTTP 응답 본문을 임시 파일로 저장하고 파일 경로를 반환합니다.
// imageURL은 파일 확장자를 추출하는 데 사용됩니다.
func SaveResponseToFile(resp *http.Response, imageURL string) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("HTTP 응답이 nil입니다")
	}

	defer resp.Body.Close()

	// 파일 확장자 추출
	ext := filepath.Ext(imageURL)
	if ext == "" {
		// URL에서 쿼리 파라미터 제거 후 확장자 추출 재시도
		parts := strings.Split(imageURL, "?")
		if len(parts) > 0 {
			ext = filepath.Ext(parts[0])
		}

		// 여전히 확장자가 없으면 기본값 설정
		if ext == "" {
			ext = ".jpg"
		}
	}

	// 유니크한 파일명 생성
	tempFileName := uuid.New().String() + ext
	tempFilePath := filepath.Join(os.TempDir(), tempFileName)

	// 이미지 데이터를 메모리에 먼저 로드 (컨텍스트 취소 문제 방지)
	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("이미지 데이터 읽기 실패: %v", err)
	}

	// 파일 생성 및 저장
	file, err := os.Create(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("임시 파일 생성 실패: %v", err)
	}
	defer file.Close()

	// 메모리에서 파일로 복사
	if _, err = file.Write(imgData); err != nil {
		return "", fmt.Errorf("이미지 저장 실패: %v", err)
	}

	return tempFilePath, nil
}
