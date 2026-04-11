package utils

import (
	"bytes"
	"image"
	"image/jpeg"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/nfnt/resize"
)

// Resize изменяет размер изображения и возвращает JPEG
func Resize(data []byte, width, height int) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	_ = format // можно использовать при необходимости

	resized := resize.Resize(uint(width), uint(height), img, resize.Lanczos3)

	buf := new(bytes.Buffer)

	// всегда сохраняем как jpeg
	if err := jpeg.Encode(buf, resized, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
