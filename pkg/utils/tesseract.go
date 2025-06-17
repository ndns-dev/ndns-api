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
	defaultPsm := "6"
	if len(psm) > 0 && psm[0] != "" {
		defaultPsm = psm[0]
	}

	cmd := exec.CommandContext(ctx, "tesseract", imagePath, "stdout", "-l", "kor", "--psm", defaultPsm, "--oem", "3", "-c", "preserve_interword_spaces=1")

	output, err := cmd.CombinedOutput()

	// 컨텍스트 취소 확인
	if ctx.Err() != nil {
		fmt.Printf("Tesseract OCR 타임아웃: %v\n", ctx.Err())
		return ""
	}

	if err != nil {
		fmt.Printf("Tesseract OCR 실행 오류: %v\n", err)
		return ""
	}

	textResult := strings.TrimSpace(string(output))

	if textResult == "" || strings.Contains(textResult, "Estimating") {
		fmt.Printf("Tesseract OCR 인식 불가: 결과 없음\n")
		return ""
	}

	return textResult
}
