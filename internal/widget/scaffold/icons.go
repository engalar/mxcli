package scaffold

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

type IconRenderer struct{}

func (IconRenderer) Render(spec Spec) []File {
	raw := minimalPNG()
	files := []string{
		spec.Name + ".icon.png",
		spec.Name + ".icon.dark.png",
		spec.Name + ".tile.png",
		spec.Name + ".tile.dark.png",
	}
	var result []File
	for _, f := range files {
		result = append(result, File{
			Path:    "src/" + f,
			Content: raw,
			Binary:  true,
		})
	}
	return result
}

func minimalPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.Transparent)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
