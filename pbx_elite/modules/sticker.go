package modules

// PBX Elite — Sticker Module
// ✅ .sticker — convert replied image/video to sticker
// ✅ Uses ffmpeg + webpmux for conversion

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// .sticker
func stickerHandler(
	v *events.Message,
	args []string,
) error {

	quoted := GetQuotedMessage(v)
	if quoted == nil {
		return ReplyErr(v, "❌ Reply to an image or video to make a sticker.")
	}

	wac := client.Get()

	// IMAGE → STICKER
	if quoted.ImageMessage != nil {
		return makeImageSticker(v, wac, quoted.ImageMessage)
	}

	// VIDEO → ANIMATED STICKER
	if quoted.VideoMessage != nil {
		return makeVideoSticker(v, wac, quoted.VideoMessage)
	}

	return ReplyErr(v, "❌ Reply to an image or video.")
}

func makeImageSticker(
	v *events.Message,
	wac *whatsmeow.Client,
	img *waProto.ImageMessage,
) error {

	data, err := wac.Download(context.Background(), img)
	if err != nil {
		return ReplyErr(v, "❌ Failed to download image.")
	}

	// Write to temp file
	tmpIn, _ := os.CreateTemp("", "sticker_in_*.jpg")
	tmpOut, _ := os.CreateTemp("", "sticker_out_*.webp")

	defer os.Remove(tmpIn.Name())
	defer os.Remove(tmpOut.Name())

	tmpIn.Write(data)
	tmpIn.Close()
	tmpOut.Close()

	// Convert using ffmpeg
	cmd := exec.Command(
		"ffmpeg", "-y",
		"-i", tmpIn.Name(),
		"-vf", "scale=512:512:force_original_aspect_ratio=decrease,pad=512:512:(ow-iw)/2:(oh-ih)/2:white@0",
		"-quality", "80",
		tmpOut.Name(),
	)

	if err := cmd.Run(); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ ffmpeg error: %v\nMake sure ffmpeg is installed.", err))
	}

	webpData, err := os.ReadFile(tmpOut.Name())
	if err != nil {
		return ReplyErr(v, "❌ Failed to read converted sticker.")
	}

	return sendSticker(v, wac, webpData)
}

func makeVideoSticker(
	v *events.Message,
	wac *whatsmeow.Client,
	vid *waProto.VideoMessage,
) error {

	data, err := wac.Download(context.Background(), vid)
	if err != nil {
		return ReplyErr(v, "❌ Failed to download video.")
	}

	tmpIn, _ := os.CreateTemp("", "sticker_in_*.mp4")
	tmpOut, _ := os.CreateTemp("", "sticker_out_*.webp")

	defer os.Remove(tmpIn.Name())
	defer os.Remove(tmpOut.Name())

	tmpIn.Write(data)
	tmpIn.Close()
	tmpOut.Close()

	// Convert to animated webp (max 3 seconds)
	cmd := exec.Command(
		"ffmpeg", "-y",
		"-i", tmpIn.Name(),
		"-t", "3",
		"-vf", "scale=512:512:force_original_aspect_ratio=decrease,fps=15",
		"-loop", "0",
		"-quality", "70",
		tmpOut.Name(),
	)

	if err := cmd.Run(); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ ffmpeg error: %v\nMake sure ffmpeg is installed.", err))
	}

	webpData, err := os.ReadFile(tmpOut.Name())
	if err != nil {
		return ReplyErr(v, "❌ Failed to read converted sticker.")
	}

	return sendSticker(v, wac, webpData)
}

func sendSticker(
	v *events.Message,
	wac *whatsmeow.Client,
	webpData []byte,
) error {

	uploaded, err := wac.Upload(
		context.Background(),
		webpData,
		whatsmeow.MediaImage,
	)
	if err != nil {
		return ReplyErr(v, "❌ Failed to upload sticker.")
	}

	mimeType := "image/webp"

	_, err = wac.SendMessage(
		context.Background(),
		v.Info.Chat,
		&waProto.Message{
			StickerMessage: &waProto.StickerMessage{
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				Mimetype:      &mimeType,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				StickerSentTS: int64Ptr(0),
				IsAnimated:    boolPtr(false),
				PngThumbnail:  webpData[:min(len(webpData), 100)],
				IsAvatar:      boolPtr(false),
			},
		},
	)

	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// REGISTER
func init() {

	Register(ModuleInfo{
		Name:        "Sticker",
		Description: "Convert image/video to sticker",

		Commands: []CommandInfo{

			{
				Pattern: "sticker",
				Handler: stickerHandler,
				Sudo:    true,
			},

			{
				Pattern: "s",
				Handler: stickerHandler,
				Sudo:    true,
			},
		},
	})
}
