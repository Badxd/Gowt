package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/types/events"
)

// .save    — save quoted/replied media to local disk
// .saved   — list saved files
// .delsave <filename> — delete a saved file

const saveDir = "./saved_media"

func ensureSaveDir() error {
	return os.MkdirAll(saveDir, 0755)
}

func saveHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// try quoted message first
	var mediaData []byte
	var ext string
	var fileName string

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// check quoted image
	if img := v.Message.GetImageMessage(); img != nil {
		data, err := wac.Download(ctx, img)
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
		}
		mediaData = data
		ext = "jpg"
	} else if vid := v.Message.GetVideoMessage(); vid != nil {
		data, err := wac.Download(ctx, vid)
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
		}
		mediaData = data
		ext = "mp4"
	} else if aud := v.Message.GetAudioMessage(); aud != nil {
		data, err := wac.Download(ctx, aud)
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
		}
		mediaData = data
		ext = "ogg"
	} else if doc := v.Message.GetDocumentMessage(); doc != nil {
		data, err := wac.Download(ctx, doc)
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
		}
		mediaData = data
		if doc.GetFileName() != "" {
			fileName = doc.GetFileName()
		}
		ext = "bin"
	} else if stk := v.Message.GetStickerMessage(); stk != nil {
		data, err := wac.Download(ctx, stk)
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
		}
		mediaData = data
		ext = "webp"
	}

	// try quoted
	if mediaData == nil {
		ext2 := v.Message.GetExtendedTextMessage()
		if ext2 != nil {
			ci := ext2.GetContextInfo()
			if ci != nil {
				qm := ci.GetQuotedMessage()
				if qm != nil {
					if img := qm.GetImageMessage(); img != nil {
						data, err := wac.Download(ctx, img)
						if err != nil {
							return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
						}
						mediaData = data
						ext = "jpg"
					} else if vid := qm.GetVideoMessage(); vid != nil {
						data, err := wac.Download(ctx, vid)
						if err != nil {
							return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
						}
						mediaData = data
						ext = "mp4"
					} else if doc := qm.GetDocumentMessage(); doc != nil {
						data, err := wac.Download(ctx, doc)
						if err != nil {
							return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
						}
						mediaData = data
						if doc.GetFileName() != "" {
							fileName = doc.GetFileName()
						}
						ext = "bin"
					}
				}
			}
		}
	}

	if mediaData == nil {
		return ReplyErr(v, "❌ No media found. Reply to an image, video, audio, document or sticker.")
	}

	if err := ensureSaveDir(); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not create save directory: %v", err))
	}

	if fileName == "" {
		fileName = fmt.Sprintf("media_%d.%s", time.Now().Unix(), ext)
	}

	// custom name from args
	if len(args) > 0 {
		name := strings.Join(args, "_")
		// preserve extension
		if !strings.Contains(name, ".") {
			name = name + "." + ext
		}
		fileName = name
	}

	savePath := filepath.Join(saveDir, fileName)
	if err := os.WriteFile(savePath, mediaData, 0644); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not save file: %v", err))
	}

	return ReplyErr(v, fmt.Sprintf("✅ Saved as `%s`\n📦 Size: %s", fileName, formatBytes(uint64(len(mediaData)))))
}

func savedListHandler(v *events.Message, args []string) error {
	if err := ensureSaveDir(); err != nil {
		return ReplyErr(v, "❌ Save directory not found.")
	}

	entries, err := os.ReadDir(saveDir)
	if err != nil || len(entries) == 0 {
		return ReplyErr(v, "📂 No saved files yet. Use `.save` while replying to media.")
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("📂 *Saved Files (%d)*", len(entries)))
	lines = append(lines, "")

	for i, e := range entries {
		info, _ := e.Info()
		size := ""
		if info != nil {
			size = formatBytes(uint64(info.Size()))
		}
		lines = append(lines, fmt.Sprintf("%d. `%s` — %s", i+1, e.Name(), size))
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func delSaveHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .delsave <filename>")
	}

	name := strings.Join(args, " ")
	// prevent directory traversal
	name = filepath.Base(name)
	path := filepath.Join(saveDir, name)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ReplyErr(v, fmt.Sprintf("❌ File not found: `%s`", name))
	}

	if err := os.Remove(path); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not delete: %v", err))
	}

	return ReplyErr(v, fmt.Sprintf("🗑️ Deleted `%s`", name))
}

func init() {
	Register(ModuleInfo{
		Name:        "Save File",
		Description: "Save media messages to local disk",
		Commands: []CommandInfo{
			{Pattern: "save", Handler: saveHandler, Sudo: true},
			{Pattern: "saved", Handler: savedListHandler, Sudo: true},
			{Pattern: "delsave", Handler: delSaveHandler, Sudo: true},
		},
	})
}
