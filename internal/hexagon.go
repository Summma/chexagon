package internal

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
	"math"
)

var whitePixel *ebiten.Image

func init() {
	whitePixel = ebiten.NewImage(1, 1)
	whitePixel.Fill(color.White)
}

func drawFilledPath(screen *ebiten.Image, path *vector.Path, clr color.Color) {
	vertices, indices := path.AppendVerticesAndIndicesForFilling(nil, nil)

	r, g, b, a := clr.RGBA()
	for i := range vertices {
		vertices[i].ColorR = float32(r) / 0xffff
		vertices[i].ColorG = float32(g) / 0xffff
		vertices[i].ColorB = float32(b) / 0xffff
		vertices[i].ColorA = float32(a) / 0xffff
	}

	screen.DrawTriangles(vertices, indices, whitePixel, nil)
}

func DrawHexagon(screen *ebiten.Image, p Position, sideLength, ox, oy float64, clr color.Color) {
	var path vector.Path
	centerX, centerY := p.PositionToCoordinate(sideLength, ox, oy)

	for i := 0.0; i < 6; i++ {
		angle := -math.Pi / 3 * i
		vx := centerX + sideLength*math.Cos(angle)
		vy := centerY + sideLength*math.Sin(angle)

		if i == 0 {
			path.MoveTo(float32(vx), float32(vy))
		} else {
			path.LineTo(float32(vx), float32(vy))
		}
	}
	path.Close()
	drawFilledPath(screen, &path, clr)
}

func DrawHexagonAtCoord(screen *ebiten.Image, centerX, centerY, sideLength float64, clr color.Color) {
	var path vector.Path

	for i := 0.0; i < 6; i++ {
		angle := -math.Pi / 3 * i
		vx := centerX + sideLength*math.Cos(angle)
		vy := centerY + sideLength*math.Sin(angle)

		if i == 0 {
			path.MoveTo(float32(vx), float32(vy))
		} else {
			path.LineTo(float32(vx), float32(vy))
		}
	}
	path.Close()
	drawFilledPath(screen, &path, clr)
}

func DrawCircle(screen *ebiten.Image, p Position, radius, ox, oy, sideLength float64, clr color.Color) {
	centerX, centerY := p.PositionToCoordinate(sideLength, ox, oy)
	DrawCircleAtCoord(screen, centerX, centerY, radius, clr)
}

func DrawCircleAtCoord(screen *ebiten.Image, centerX, centerY, radius float64, clr color.Color) {
	var path vector.Path

	segments := 24
	for i := 0; i <= segments; i++ {
		angle := 2 * math.Pi * float64(i) / float64(segments)
		vx := centerX + radius*math.Cos(angle)
		vy := centerY + radius*math.Sin(angle)

		if i == 0 {
			path.MoveTo(float32(vx), float32(vy))
		} else {
			path.LineTo(float32(vx), float32(vy))
		}
	}
	path.Close()
	drawFilledPath(screen, &path, clr)
}

func DrawRing(screen *ebiten.Image, p Position, outerRadius, thickness, ox, oy, sideLength float64, clr color.Color) {
	centerX, centerY := p.PositionToCoordinate(sideLength, ox, oy)
	DrawRingAtCoord(screen, centerX, centerY, outerRadius, thickness, clr)
}

func DrawRingAtCoord(screen *ebiten.Image, centerX, centerY, outerRadius, thickness float64, clr color.Color) {
	innerRadius := outerRadius - thickness
	if innerRadius < 0 {
		innerRadius = 0
	}

	segments := 32

	for i := 0; i < segments; i++ {
		angle1 := 2 * math.Pi * float64(i) / float64(segments)
		angle2 := 2 * math.Pi * float64(i+1) / float64(segments)

		ox1 := centerX + outerRadius*math.Cos(angle1)
		oy1 := centerY + outerRadius*math.Sin(angle1)
		ox2 := centerX + outerRadius*math.Cos(angle2)
		oy2 := centerY + outerRadius*math.Sin(angle2)

		ix1 := centerX + innerRadius*math.Cos(angle1)
		iy1 := centerY + innerRadius*math.Sin(angle1)
		ix2 := centerX + innerRadius*math.Cos(angle2)
		iy2 := centerY + innerRadius*math.Sin(angle2)

		var path vector.Path
		path.MoveTo(float32(ox1), float32(oy1))
		path.LineTo(float32(ox2), float32(oy2))
		path.LineTo(float32(ix2), float32(iy2))
		path.LineTo(float32(ix1), float32(iy1))
		path.Close()

		drawFilledPath(screen, &path, clr)
	}
}
