package internal

import (
	"bytes"
	"embed"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

var embeddedAssets embed.FS

var imageCache = make(map[string]*ebiten.Image)

func SetAssets(fs embed.FS) {
	embeddedAssets = fs
}

func LoadImage(path string) (*ebiten.Image, error) {
	if img, ok := imageCache[path]; ok {
		return img, nil
	}

	data, err := embeddedAssets.ReadFile(path)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	ebitenImg := ebiten.NewImageFromImage(img)

	imageCache[path] = ebitenImg

	return ebitenImg, nil
}
