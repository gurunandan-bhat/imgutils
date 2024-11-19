package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"math"
	"os"

	"github.com/disintegration/gift"
	"github.com/muesli/smartcrop"
	"github.com/ogier/pflag"
)

type giftResizer struct {
	anchor     gift.Anchor
	resampling gift.Resampling
}

type SubImager interface {
	SubImage(image.Rectangle) image.Image
}

func (r giftResizer) Resize(img image.Image, w, h uint) image.Image {

	scaleX, scaleY := calcFactorsNfnt(w, h, float64(img.Bounds().Dx()), float64(img.Bounds().Dy()))
	if w == 0 {
		w = uint(math.Ceil(float64(img.Bounds().Dx()) / scaleX))
	}
	if h == 0 {
		h = uint(math.Ceil(float64(img.Bounds().Dy()) / scaleY))
	}

	g := gift.New(
		gift.ResizeToFill(int(w), 0, r.resampling, r.anchor),
	)
	dstImage := image.NewRGBA(g.Bounds(img.Bounds()))
	g.Draw(dstImage, img)

	return dstImage
}

func smartCrop(img image.Image, w, h int) (image.Image, error) {

	r := giftResizer{
		anchor:     gift.CenterAnchor,
		resampling: gift.LanczosResampling,
	}
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	analyzer := smartcrop.NewAnalyzerWithLogger(r, smartcrop.Logger{DebugMode: true, Log: logger})

	rect, err := analyzer.FindBestCrop(img, w, h)
	if err != nil {
		return nil, fmt.Errorf("error finding best crop: %w", err)
	}

	fmt.Printf("%v\n", rect)

	croppedImg := img.(SubImager).SubImage(rect)
	if croppedImg.Bounds().Dx() != w || croppedImg.Bounds().Dy() != h {
		croppedImg = r.Resize(croppedImg, uint(w), uint(h))
	}
	return croppedImg, nil
}

func giftCrop(img image.Image, w, h int) (image.Image, error) {

	g := gift.New(
		gift.ResizeToFill(w, h, gift.CubicResampling, gift.CenterAnchor),
	)
	dstImg := image.NewRGBA(g.Bounds(img.Bounds()))
	g.Draw(dstImg, img)

	return dstImg, nil
}

func calcFactorsNfnt(width, height uint, oldWidth, oldHeight float64) (scaleX, scaleY float64) {
	if width == 0 {
		if height == 0 {
			scaleX = 1.0
			scaleY = 1.0
		} else {
			scaleY = oldHeight / float64(height)
			scaleX = scaleY
		}
	} else {
		scaleX = oldWidth / float64(width)
		if height == 0 {
			scaleY = scaleX
		} else {
			scaleY = oldHeight / float64(height)
		}
	}
	return
}

func main() {

	var inPath, outPath string
	pflag.StringVarP(&inPath, "input-image", "i", "", "input image")
	pflag.StringVarP(&outPath, "output-image", "o", "", "output image")

	var width, height int
	pflag.IntVarP(&width, "width", "w", 0, "image width")
	pflag.IntVarP(&height, "height", "h", 0, "image height")

	pflag.Parse()

	f, err := os.Open(inPath)
	if err != nil {
		log.Fatalf("error opening file %s: %s", inPath, err)
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		log.Fatalf("error decoding image %s: %s", inPath, err)
	}
	if format != "jpeg" {
		log.Fatalf("error decoding image - expected jpeg, got: %s", format)
	}

	dstImg, err := smartCrop(img, width, height)

	outF, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("error creating file %s: %s", outPath, err)
	}
	if err = jpeg.Encode(outF, dstImg, &jpeg.Options{Quality: 85}); err != nil {
		log.Fatalf("error writing image to %s: %s", outPath, err)
	}

	outF.Close()
}
