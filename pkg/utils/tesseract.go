package utils

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func RunTesseract(imagePath string) *exec.Cmd {
	return exec.Command("tesseract", imagePath, "stdout", "-l", "kor", "--psm", "6", "--oem", "3", "-c", "preserve_interword_spaces=1")
}

func ConvertDPI(srcPath, dstPath string) error {
	cmd := exec.Command("convert", srcPath, "-units", "PixelsPerInch", "-density", "300", dstPath)
	return cmd.Run()
}

// RunTesseractWithContext는 컨텍스트와 함께 Tesseract OCR을 실행하고 결과를 반환합니다.
// 내부에서 에러를 완전히 처리하므로 항상 텍스트 결과만 반환합니다.
// 오류가 발생하면 에러 메시지를 로깅하고 빈 문자열을 반환합니다.
func RunTesseractWithContext(ctx context.Context, imagePath string, psm ...string) string {
	// psm의 기본값 설정
	defaultPsm := "6"
	if len(psm) > 0 && psm[0] != "" {
		defaultPsm = psm[0]
	}

	// Tesseract 명령 생성
	cmd := exec.CommandContext(ctx, "tesseract", imagePath, "stdout", "-l", "kor", "--psm", defaultPsm, "--oem", "3", "-c", "preserve_interword_spaces=1")

	// 결과와 에러를 받을 채널 생성
	resultCh := make(chan struct {
		output []byte
		err    error
	})

	// 비동기로 명령 실행
	go func() {
		output, err := cmd.CombinedOutput()
		select {
		case <-ctx.Done():
			// 이미 컨텍스트가 취소되었다면 결과 전송 생략
			return
		default:
			resultCh <- struct {
				output []byte
				err    error
			}{output, err}
		}
	}()

	// 타임아웃 또는 결과 대기
	select {
	case <-ctx.Done():
		fmt.Printf("Tesseract OCR 타임아웃: %v\n", ctx.Err())
		return "" // 타임아웃 시 빈 문자열 반환 (이미지 상태에 따라 타임아웃은 확인될 수 있음)
	case result := <-resultCh:
		// 기타 오류 확인
		if result.err != nil {
			fmt.Printf("Tesseract OCR 실행 오류: %v\n", result.err)
			return "" // 오류 발생 시 빈 문자열 반환
		}

		// 성공적으로 실행된 경우 결과 반환
		textResult := strings.TrimSpace(string(result.output))

		// 결과가 없는 경우
		if textResult == "" || strings.Contains(textResult, "Estimating") {
			fmt.Printf("Tesseract OCR 인식 불가: 결과 없음\n")
			return ""
		}

		return textResult
	}
}
