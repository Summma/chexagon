package internal

import (
	"image/color"
)

var (
	MoveIndicatorColor    = color.RGBA{R: 70, G: 130, B: 220, A: 200} // Blue for normal moves
	CaptureIndicatorColor = color.RGBA{R: 220, G: 70, B: 70, A: 180}  // Red for captures
)

func GreenHexPalette(intensity byte) []color.Color {
	clampedIntensity := max(min(intensity, 100), 0)
	return []color.Color{
		color.RGBA{100 - clampedIntensity, 100, 100 - clampedIntensity, 255}, // Dark
		color.RGBA{150 - clampedIntensity, 150, 150 - clampedIntensity, 255}, // Less Dark
		color.RGBA{200 - clampedIntensity, 200, 200 - clampedIntensity, 255}, // Light
	}
}

func BlueHexPalette(intensity byte) []color.Color {
	clampedIntensity := max(min(intensity, 100), 0)
	return []color.Color{
		color.RGBA{100 - clampedIntensity, 100 - clampedIntensity, 120, 255}, // Dark
		color.RGBA{140 - clampedIntensity, 140 - clampedIntensity, 170, 255}, // Less Dark
		color.RGBA{180 - clampedIntensity, 180 - clampedIntensity, 220, 255}, // Light
	}
}

func BrownHexPalette(intensity byte) []color.Color {
	clampedIntensity := max(min(intensity, 100), 0)
	return []color.Color{
		color.RGBA{139 - clampedIntensity/2, 90 - clampedIntensity/2, 43, 255},   // Dark brown
		color.RGBA{160 - clampedIntensity/2, 120 - clampedIntensity/2, 60, 255},  // Medium brown
		color.RGBA{210 - clampedIntensity/2, 180 - clampedIntensity/2, 140, 255}, // Light tan
	}
}

func GrayHexPalette(intensity byte) []color.Color {
	clampedIntensity := max(min(intensity, 100), 0)
	return []color.Color{
		color.RGBA{80 - clampedIntensity/2, 80 - clampedIntensity/2, 80 - clampedIntensity/2, 255},    // Dark
		color.RGBA{130 - clampedIntensity/2, 130 - clampedIntensity/2, 130 - clampedIntensity/2, 255}, // Medium
		color.RGBA{180 - clampedIntensity/2, 180 - clampedIntensity/2, 180 - clampedIntensity/2, 255}, // Light
	}
}

func PurpleHexPalette(intensity byte) []color.Color {
	clampedIntensity := max(min(intensity, 100), 0)
	return []color.Color{
		color.RGBA{100 - clampedIntensity/2, 70 - clampedIntensity/2, 110, 255},  // Dark purple
		color.RGBA{140 - clampedIntensity/2, 100 - clampedIntensity/2, 150, 255}, // Medium purple
		color.RGBA{190 - clampedIntensity/2, 160 - clampedIntensity/2, 200, 255}, // Light purple
	}
}

func TealHexPalette(intensity byte) []color.Color {
	clampedIntensity := max(min(intensity, 100), 0)
	return []color.Color{
		color.RGBA{50 - clampedIntensity/2, 100 - clampedIntensity/2, 100, 255},  // Dark teal
		color.RGBA{70 - clampedIntensity/2, 140 - clampedIntensity/2, 140, 255},  // Medium teal
		color.RGBA{100 - clampedIntensity/2, 190 - clampedIntensity/2, 190, 255}, // Light teal
	}
}

func CoralHexPalette(intensity byte) []color.Color {
	clampedIntensity := max(min(intensity, 100), 0)
	return []color.Color{
		color.RGBA{180 - clampedIntensity/2, 100 - clampedIntensity/2, 80, 255},  // Dark coral
		color.RGBA{210 - clampedIntensity/2, 130 - clampedIntensity/2, 100, 255}, // Medium coral
		color.RGBA{240 - clampedIntensity/2, 170 - clampedIntensity/2, 140, 255}, // Light coral
	}
}

func MintHexPalette(intensity byte) []color.Color {
	clampedIntensity := max(min(intensity, 100), 0)
	return []color.Color{
		color.RGBA{60 - clampedIntensity/2, 120 - clampedIntensity/2, 100, 255},  // Dark mint
		color.RGBA{100 - clampedIntensity/2, 170 - clampedIntensity/2, 140, 255}, // Medium mint
		color.RGBA{150 - clampedIntensity/2, 210 - clampedIntensity/2, 180, 255}, // Light mint
	}
}

func SlateHexPalette(intensity byte) []color.Color {
	clampedIntensity := max(min(intensity, 100), 0)
	return []color.Color{
		color.RGBA{70 - clampedIntensity/2, 80 - clampedIntensity/2, 100, 255},   // Dark slate
		color.RGBA{110 - clampedIntensity/2, 120 - clampedIntensity/2, 140, 255}, // Medium slate
		color.RGBA{160 - clampedIntensity/2, 170 - clampedIntensity/2, 190, 255}, // Light slate
	}
}

func RoseHexPalette(intensity byte) []color.Color {
	clampedIntensity := max(min(intensity, 100), 0)
	return []color.Color{
		color.RGBA{140 - clampedIntensity/2, 80 - clampedIntensity/2, 100, 255},  // Dark rose
		color.RGBA{180 - clampedIntensity/2, 120 - clampedIntensity/2, 140, 255}, // Medium rose
		color.RGBA{220 - clampedIntensity/2, 170 - clampedIntensity/2, 190, 255}, // Light rose
	}
}
