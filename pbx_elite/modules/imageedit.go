package modules

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math/rand"
	"strings"

	"PbxGb/client"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// .invert  — invert image colors
// .grey    — convert image to greyscale
// .blur    — simple box blur (3x3)
// .pixelate — pixelate image (8x8 blocks)
// .noise   — add random noise to image

func downloadImageFromMsg(v *events.Message) ([]byte, string, error) {
	wac := client.Get()

	if img := v.Message.GetImageMessage(); img != nil {
		data, err := wac.Download(img)
		return data, img.GetMimetype(), err
	}
	// Check quoted
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		if ci := ext.GetContextInfo(); ci != nil {
			if qm := ci.GetQuotedMessage(); qm != nil {
				if qi := qm.GetImageMessage(); qi != nil {
					data, err := wac.Download(qi)
					return data, qi.GetMimetype(), err
				}
			}
		}
	}
	return nil, "", fmt.Errorf("no image found")
}

func decodeImage(data []byte, mime string) (image.Image, error) {
	r := bytes.NewReader(data)
	if strings.Contains(mime, "png") {
		return png.Decode(r)
	}
	return jpeg.Decode(r)
}

func encodeImagePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sendProcessedImage(v *events.Message, imgBytes []byte, caption string) error {
	wac := client.Get()
	uploaded, err := wac.Upload(context.Background(), imgBytes, whatsmeow.MediaImage)
	if err != nil {
		return err
	}
	imgMsg := &waE2E.ImageMessage{
		URL:           &uploaded.URL,
		DirectPath:    &uploaded.DirectPath,
		MediaKey:      uploaded.MediaKey,
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    &uploaded.FileLength,
		Mimetype:      strPtr("image/png"),
		Caption:       &caption,
	}
	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waE2E.Message{ImageMessage: imgMsg})
	return err
}

// Invert colors
func invertImage(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			dst.Set(x, y, color.RGBA{
				R: uint8(255 - r>>8),
				G: uint8(255 - g>>8),
				B: uint8(255 - b>>8),
				A: uint8(a >> 8),
			})
		}
	}
	return dst
}

// Greyscale
func greyscaleImage(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			grey := uint8((19595*r>>8 + 38470*g>>8 + 7471*b>>8) >> 16)
			dst.Set(x, y, color.RGBA{R: grey, G: grey, B: grey, A: uint8(a >> 8)})
		}
	}
	return dst
}

// Pixelate (block size 12)
func pixelateImage(src image.Image, blockSize int) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += blockSize {
		for x := bounds.Min.X; x < bounds.Max.X; x += blockSize {
			// Average color of block
			var rSum, gSum, bSum, aSum, count uint64
			for by := y; by < y+blockSize && by < bounds.Max.Y; by++ {
				for bx := x; bx < x+blockSize && bx < bounds.Max.X; bx++ {
					r, g, b, a := src.At(bx, by).RGBA()
					rSum += uint64(r >> 8)
					gSum += uint64(g >> 8)
					bSum += uint64(b >> 8)
					aSum += uint64(a >> 8)
					count++
				}
			}
			if count == 0 {
				continue
			}
			avg := color.RGBA{
				R: uint8(rSum / count),
				G: uint8(gSum / count),
				B: uint8(bSum / count),
				A: uint8(aSum / count),
			}
			for by := y; by < y+blockSize && by < bounds.Max.Y; by++ {
				for bx := x; bx < x+blockSize && bx < bounds.Max.X; bx++ {
					dst.Set(bx, by, avg)
				}
			}
		}
	}
	return dst
}

// Add noise
func noiseImage(src image.Image, intensity int) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			n := rand.Intn(intensity*2+1) - intensity
			clamp := func(v int) uint8 {
				if v < 0 {
					return 0
				}
				if v > 255 {
					return 255
				}
				return uint8(v)
			}
			dst.Set(x, y, color.RGBA{
				R: clamp(int(r>>8) + n),
				G: clamp(int(g>>8) + n),
				B: clamp(int(b>>8) + n),
				A: uint8(a >> 8),
			})
		}
	}
	return dst
}

func makeImageEditHandler(label string, transform func(image.Image) image.Image) CommandHandler {
	return func(v *events.Message, args []string) error {
		data, mime, err := downloadImageFromMsg(v)
		if err != nil {
			return ReplyErr(v, "Reply to an image with ."+strings.ToLower(label))
		}
		img, err := decodeImage(data, mime)
		if err != nil {
			return ReplyErr(v, "❌ Could not decode image: "+err.Error())
		}
		out := transform(img)
		outBytes, err := encodeImagePNG(out)
		if err != nil {
			return ReplyErr(v, "❌ Encode error: "+err.Error())
		}
		return sendProcessedImage(v, outBytes, "🎨 "+label)
	}
}

func pixelateHandler(v *events.Message, args []string) error {
	data, mime, err := downloadImageFromMsg(v)
	if err != nil {
		return ReplyErr(v, "Reply to an image with .pixelate")
	}
	img, err := decodeImage(data, mime)
	if err != nil {
		return ReplyErr(v, "❌ Could not decode image: "+err.Error())
	}
	out := pixelateImage(img, 12)
	outBytes, err := encodeImagePNG(out)
	if err != nil {
		return ReplyErr(v, "❌ Encode error: "+err.Error())
	}
	return sendProcessedImage(v, outBytes, "🟦 Pixelated")
}

func noiseHandler(v *events.Message, args []string) error {
	data, mime, err := downloadImageFromMsg(v)
	if err != nil {
		return ReplyErr(v, "Reply to an image with .noise")
	}
	img, err := decodeImage(data, mime)
	if err != nil {
		return ReplyErr(v, "❌ Could not decode image: "+err.Error())
	}
	out := noiseImage(img, 40)
	outBytes, err := encodeImagePNG(out)
	if err != nil {
		return ReplyErr(v, "❌ Encode error: "+err.Error())
	}
	return sendProcessedImage(v, outBytes, "📡 Noisy")
}

func init() {
	Register(ModuleInfo{
		Name:        "ImageEdit",
		Description: "Image effects: invert, greyscale, pixelate, noise",
		Commands: []CommandInfo{
			{Pattern: "invert", Handler: makeImageEditHandler("Inverted", invertImage), Sudo: true},
			{Pattern: "grey", Handler: makeImageEditHandler("Greyscale", greyscaleImage), Sudo: true},
			{Pattern: "pixelate", Handler: pixelateHandler, Sudo: true},
			{Pattern: "noise", Handler: noiseHandler, Sudo: true},
		},
	})
}
