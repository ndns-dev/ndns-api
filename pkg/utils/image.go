package utils

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/gif" // GIF 포맷 지원
	"image/jpeg"
	_ "image/jpeg" // JPEG 포맷 지원
	_ "image/png"  // PNG 포맷 지원
	"os"
)

// ImageDimensions는 이미지의 가로/세로 크기 정보를 담고 있습니다
type ImageDimensions struct {
	Width  int
	Height int
}

// GetImageDimensions는 이미지 파일의 가로/세로 크기를 반환합니다
func GetImageDimensions(filePath string) (*ImageDimensions, error) {
	// 파일 열기
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("파일 열기 실패: %v", err)
	}
	defer file.Close()

	// 표준 image 패키지를 사용하여 이미지 디코딩
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("이미지 디코딩 실패: %v", err)
	}

	// 이미지 크기 반환
	bounds := img.Bounds()
	return &ImageDimensions{
		Width:  bounds.Max.X - bounds.Min.X,
		Height: bounds.Max.Y - bounds.Min.Y,
	}, nil
}

// CropImageTop은 큰 이미지를 상단 일부만 잘라서 새 파일로 저장합니다
func CropImageTop(sourcePath string, maxHeight int) (string, error) {
	// 원본 파일 열기
	file, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("원본 파일 열기 실패: %v", err)
	}
	defer file.Close()

	// 이미지 디코딩
	sourceImg, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("이미지 디코딩 실패: %v", err)
	}

	// 원본 이미지 크기 확인
	bounds := sourceImg.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// 이미지가 이미 maxHeight 이하면 원본 반환
	if height <= maxHeight {
		return sourcePath, nil
	}

	// 잘라낼 영역 계산 (상단 maxHeight 픽셀)
	cropRect := image.Rect(0, 0, width, maxHeight)

	// 새 이미지 생성
	croppedImg := image.NewRGBA(cropRect)

	// 원본 이미지에서 상단 부분만 새 이미지로 복사
	draw.Draw(croppedImg, cropRect, sourceImg, bounds.Min, draw.Src)

	// 새 파일 이름 생성 (원본 파일명_cropped)
	croppedPath := sourcePath + "_cropped.jpg"

	// 새 파일 생성
	outFile, err := os.Create(croppedPath)
	if err != nil {
		return "", fmt.Errorf("출력 파일 생성 실패: %v", err)
	}
	defer outFile.Close()

	// 이미지를 JPEG 형식으로 저장 (간단함을 위해 JPEG만 지원)
	err = jpeg.Encode(outFile, croppedImg, &jpeg.Options{Quality: 90})
	if err != nil {
		return "", fmt.Errorf("이미지 인코딩 실패: %v", err)
	}

	return croppedPath, nil
}

// IsGifImage는 파일이 GIF 이미지인지 확인합니다 (파일 시그니처 검사)
func IsGifImage(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// GIF 파일 시그니처 확인 (GIF87a 또는 GIF89a)
	header := make([]byte, 6)
	if _, err := file.Read(header); err != nil {
		return false
	}

	return string(header) == "GIF87a" || string(header) == "GIF89a"
}
