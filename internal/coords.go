package internal

import (
	"math"
)

type Axis struct {
	x, y, z int
}

type Direction int

type Vector struct {
	Dir Direction
	Mag int
}

const (
	DirUp Direction = iota
	DirDown
	DirUpLeft
	DirDownLeft
	DirUpRight
	DirDownRight
)

var directionVectors = [6]Axis{
	{0, 1, -1}, // UP
	{0, -1, 1}, // DOWN
	{-1, 1, 0}, // UP LEFT
	{-1, 0, 1}, // DOWN LEFT
	{1, 0, -1}, // UP RIGHT
	{1, -1, 0}, // DOWN RIGHT
}

func (d Direction) Axis() Axis {
	return directionVectors[d]
}

type Position struct {
	X, Y, Z int
}

func (p Position) Equals(o Position) bool {
	return p.X == o.X && p.Y == o.Y && p.Z == o.Z
}

func (p Position) PositionToCoordinate(sideLength, ox, oy float64) (float64, float64) {
	x := sideLength * (3.0 / 2.0 * float64(p.X))
	y := sideLength * (math.Sqrt(3)/2*float64(p.X) + math.Sqrt(3)*float64(p.Z))
	return x + ox, y + oy
}

func CoordinateToPosition(sideLength, ox, oy, x, y float64) Position {
	x -= ox
	y -= oy

	px := (2 * x) / (3 * sideLength)
	pz := y/(sideLength*math.Sqrt(3)) - px/2
	py := -px - pz

	rx, ry, rz := math.Round(px), math.Round(py), math.Round(pz)

	dx := math.Abs(rx - px)
	dy := math.Abs(ry - py)
	dz := math.Abs(rz - pz)

	if dx > dy && dx > dz {
		rx = -ry - rz
	} else if dy > dz {
		ry = -rx - rz
	} else {
		rz = -rx - ry
	}

	return Position{
		X: int(rx),
		Y: int(ry),
		Z: int(rz),
	}
}

func (p *Position) Add(dir Direction) {
	axis := dir.Axis()

	p.X += axis.x
	p.Y += axis.y
	p.Z += axis.z
}

func (p *Position) AddVector(v Vector) {
	axis := v.Dir.Axis()

	p.X += axis.x * v.Mag
	p.Y += axis.y * v.Mag
	p.Z += axis.z * v.Mag
}

func (p *Position) AddVectors(vectors ...Vector) {
	for _, vector := range vectors {
		p.AddVector(vector)
	}
}

func (p *Position) AddReturn(dir Direction) Position {
	var o Position
	o.X = p.X
	o.Y = p.Y
	o.Z = p.Z

	o.Add(dir)

	return o
}

func (p *Position) AddVectorReturn(v Vector) Position {
	var o Position
	o.X = p.X
	o.Y = p.Y
	o.Z = p.Z

	o.AddVector(v)

	return o
}

func (p *Position) AddVectorsReturn(vectors ...Vector) Position {
	var o Position
	o.X = p.X
	o.Y = p.Y
	o.Z = p.Z

	for _, vector := range vectors {
		o.AddVector(vector)
	}

	return o
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func inBoardSlow(p Position) bool {
	radius := 5
	ax, ay, az := p.X, p.Y, p.Z
	if ax < 0 { ax = -ax }
	if ay < 0 { ay = -ay }
	if az < 0 { az = -az }
	return ax <= radius && ay <= radius && az <= radius
}

func (p Position) InBoard() bool {
	x := p.X + 5
	y := p.Y + 5
	if x < 0 || x > 10 || y < 0 || y > 10 {
		return false
	}
	return inBoardTable[x][y]
}

var posToIdxTable [11][11]int

var inBoardTable [11][11]bool

func init() {
	for i := 0; i < 11; i++ {
		for j := 0; j < 11; j++ {
			posToIdxTable[i][j] = -1
		}
	}
}

func InitPosToIdxTable() {
	for pos, idx := range positionToIndex {
		posToIdxTable[pos.X+5][pos.Y+5] = idx
		inBoardTable[pos.X+5][pos.Y+5] = true
	}
}

func fastPosToIdx(pos Position) int {
	x := pos.X + 5
	y := pos.Y + 5
	if x < 0 || x > 10 || y < 0 || y > 10 {
		return -1
	}
	return posToIdxTable[x][y]
}

func PositionToIndex(p Position) int {
	return fastPosToIdx(p)
}

func GetPiece(board *Board, pos Position) *Piece {
	idx := fastPosToIdx(pos)
	if idx < 0 {
		return nil
	}
	return board[idx]
}

func SetPiece(board *Board, pos Position, piece *Piece) {
	idx := fastPosToIdx(pos)
	if idx >= 0 {
		board[idx] = piece
	}
}

func IndexToPosition(index int) Position {
	if index < 0 || index >= len(indexToPosition) {
		return Position{}
	}
	return indexToPosition[index]
}
