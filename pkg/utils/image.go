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

func Resize(data []byte, width, height int) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	_ = format

	resized := resize.Resize(uint(width), uint(height), img, resize.Lanczos3)

	buf := new(bytes.Buffer)

	if err := jpeg.Encode(buf, resized, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
