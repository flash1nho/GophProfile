package utils

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

var ErrInvalidFormat = errors.New("invalid format")

type ImageFormat string

const (
	FormatJPEG ImageFormat = "jpeg"
	FormatPNG  ImageFormat = "png"
	FormatWEBP ImageFormat = "webp"
)

type ImageSize string

const (
	Size100      ImageSize = "100x100"
	Size300      ImageSize = "300x300"
	SizeOriginal ImageSize = "original"
)

func Decode(data []byte) (image.Image, string, error) {
	return image.Decode(bytes.NewReader(data))
}

func Resize(img image.Image, size ImageSize) image.Image {
	switch size {
	case Size100:
		return imaging.Fill(img, 100, 100, imaging.Center, imaging.Lanczos)
	case Size300:
		return imaging.Fill(img, 300, 300, imaging.Center, imaging.Lanczos)
	default:
		return img
	}
}
func flattenToWhite(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	bg := image.NewRGBA(bounds)

	draw.Draw(bg, bounds, &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(bg, bounds, img, bounds.Min, draw.Over)

	return bg
}

func Encode(img image.Image, format ImageFormat) ([]byte, string, error) {
	buf := new(bytes.Buffer)

	switch format {
	case FormatJPEG:
		img = flattenToWhite(img)
		err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 90})
		return buf.Bytes(), "image/jpeg", err

	case FormatPNG:
		err := png.Encode(buf, img)
		return buf.Bytes(), "image/png", err

	case FormatWEBP:
		err := webp.Encode(buf, img, &webp.Options{
			Lossless: false,
			Quality:  90,
		})
		return buf.Bytes(), "image/webp", err

	default:
		return nil, "", ErrInvalidFormat
	}
}

func Process(data []byte, size ImageSize, format ImageFormat) ([]byte, string, error) {
	img, _, err := Decode(data)
	if err != nil {
		return nil, "", err
	}

	img = Resize(img, size)

	return Encode(img, format)
}

func ResizeBytes(data []byte, width, height int) ([]byte, error) {
	img, _, err := Decode(data)
	if err != nil {
		return nil, err
	}

	resized := imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func GetDimensions(data []byte) (int, int, error) {
	img, _, err := Decode(data)
	if err != nil {
		return 0, 0, err
	}

	b := img.Bounds()
	return b.Dx(), b.Dy(), nil
}
