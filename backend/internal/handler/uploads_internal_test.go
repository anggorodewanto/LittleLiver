package handler

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func makeRGBJPEGInternal(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func makeRGBPNGInternal(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestImagemagickThumbnail_ResizesLargeJPEGToAround300pxWide(t *testing.T) {
	src := makeRGBJPEGInternal(t, 1200, 900)

	thumbData, mime, err := imagemagickThumbnail(context.Background(), src, "image/jpeg")
	if err != nil {
		t.Fatalf("imagemagickThumbnail: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("mime: got %q, want image/jpeg", mime)
	}

	img, _, err := image.Decode(bytes.NewReader(thumbData))
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	w := img.Bounds().Dx()
	if w < 280 || w > 320 {
		t.Errorf("width: got %d, want ~300", w)
	}
}

func TestImagemagickThumbnail_PreservesAspectRatio(t *testing.T) {
	src := makeRGBJPEGInternal(t, 1200, 900)

	thumbData, _, err := imagemagickThumbnail(context.Background(), src, "image/jpeg")
	if err != nil {
		t.Fatalf("imagemagickThumbnail: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(thumbData))
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	expectedH := w * 900 / 1200
	if h < expectedH-5 || h > expectedH+5 {
		t.Errorf("height: got %d, want ~%d", h, expectedH)
	}
}

func TestImagemagickThumbnail_DoesNotUpscaleSmallImage(t *testing.T) {
	src := makeRGBJPEGInternal(t, 100, 80)

	thumbData, _, err := imagemagickThumbnail(context.Background(), src, "image/jpeg")
	if err != nil {
		t.Fatalf("imagemagickThumbnail: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(thumbData))
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	if img.Bounds().Dx() != 100 {
		t.Errorf("expected width 100 (no upscale), got %d", img.Bounds().Dx())
	}
}

func TestImagemagickThumbnail_PNGStaysPNG(t *testing.T) {
	src := makeRGBPNGInternal(t, 1200, 900)

	thumbData, mime, err := imagemagickThumbnail(context.Background(), src, "image/png")
	if err != nil {
		t.Fatalf("imagemagickThumbnail: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("mime: got %q, want image/png", mime)
	}

	img, _, err := image.Decode(bytes.NewReader(thumbData))
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	w := img.Bounds().Dx()
	if w < 280 || w > 320 {
		t.Errorf("width: got %d, want ~300", w)
	}
}

func TestImagemagickResizeOriginal_DownscalesLargeJPEG(t *testing.T) {
	src := makeRGBJPEGInternal(t, 4000, 3000)

	out, mime, err := imagemagickResizeOriginal(context.Background(), src, "image/jpeg")
	if err != nil {
		t.Fatalf("imagemagickResizeOriginal: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("mime: got %q, want image/jpeg", mime)
	}

	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w > 2000 || h > 2000 {
		t.Errorf("dims: got %dx%d, want longest side <=2000", w, h)
	}
	if w != 2000 {
		t.Errorf("width: got %d, want 2000 (4:3 input scaled to 2000x1500)", w)
	}
}

func TestImagemagickResizeOriginal_PassesThroughSmallImage(t *testing.T) {
	src := makeRGBJPEGInternal(t, 800, 600)

	out, _, err := imagemagickResizeOriginal(context.Background(), src, "image/jpeg")
	if err != nil {
		t.Fatalf("imagemagickResizeOriginal: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if img.Bounds().Dx() != 800 || img.Bounds().Dy() != 600 {
		t.Errorf("dims: got %dx%d, want 800x600 unchanged", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestImagemagickResizeOriginal_PNGBecomesJPEG(t *testing.T) {
	src := makeRGBPNGInternal(t, 4000, 3000)

	out, mime, err := imagemagickResizeOriginal(context.Background(), src, "image/png")
	if err != nil {
		t.Fatalf("imagemagickResizeOriginal: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("mime: got %q, want image/jpeg (PNG should be re-encoded as JPEG to save space)", mime)
	}

	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if img.Bounds().Dx() > 2000 {
		t.Errorf("width: got %d, want <=2000", img.Bounds().Dx())
	}
}
