package internal

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type PieceType int
type Color int

const (
	Pawn PieceType = iota
	Knight
	Bishop
	Rook
	Queen
	King
)

const (
	White = iota
	Black
)

var (
	rookDirs = []Direction{DirUp, DirDown, DirUpLeft, DirUpRight, DirDownLeft, DirDownRight}

	bishopDirs = [][2]Direction{
		{DirUp, DirUpRight},
		{DirDownRight, DirUpRight},
		{DirDownRight, DirDown},
		{DirDownLeft, DirDown},
		{DirDownLeft, DirUpLeft},
		{DirUp, DirUpLeft},
	}

	knightOffsets = [][2]Vector{
		{{DirUp, 2}, {DirUpLeft, 1}},
		{{DirUp, 2}, {DirUpRight, 1}},
		{{DirDown, 2}, {DirDownLeft, 1}},
		{{DirDown, 2}, {DirDownRight, 1}},
		{{DirUpLeft, 2}, {DirUp, 1}},
		{{DirUpLeft, 2}, {DirDownLeft, 1}},
		{{DirUpRight, 2}, {DirUp, 1}},
		{{DirUpRight, 2}, {DirDownRight, 1}},
		{{DirDownLeft, 2}, {DirDown, 1}},
		{{DirDownLeft, 2}, {DirUpLeft, 1}},
		{{DirDownRight, 2}, {DirDown, 1}},
		{{DirDownRight, 2}, {DirUpRight, 1}},
	}

	kingOffsets = [][]Direction{
		{DirUp, DirUpRight},
		{DirDownRight, DirUpRight},
		{DirDownRight, DirDown},
		{DirDownLeft, DirDown},
		{DirDownLeft, DirUpLeft},
		{DirUp, DirUpLeft},
		{DirUp},
		{DirDown},
		{DirUpLeft},
		{DirUpRight},
		{DirDownLeft},
		{DirDownRight},
	}
)

var pieceTypeName = map[PieceType]string{
	Pawn:   "Pawn",
	Knight: "Knight",
	Bishop: "Bishop",
	Rook:   "Rook",
	Queen:  "Queen",
	King:   "King",
}

var pieceColorName = map[Color]string{
	White: "White",
	Black: "Black",
}

func (p PieceType) String() string {
	return pieceTypeName[p]
}

func (c Color) String() string {
	return pieceColorName[c]
}

func (c Color) Other() Color {
	if c == White {
		return Black
	}
	return White
}

type Piece struct {
	Type     PieceType
	Color    Color
	Image    string
	Position Position
	Moved    bool
}



var plainMovePool = make(chan []PlainMove, 32)

func getPlainMoveBuffer() []PlainMove {
	select {
	case buf := <-plainMovePool:
		return buf[:0]
	default:
		return make([]PlainMove, 0, 32)
	}
}

func putPlainMoveBuffer(buf []PlainMove) {
	select {
	case plainMovePool <- buf:
	default:

	}
}

func GenerateAllMoves(game *Game) []PlainMove {
	moves := []PlainMove{}

	for _, piece := range game.Board {
		if piece != nil {
			moves = append(moves, piece.GenerateMoves(game)...)
		}
	}

	return moves
}

func (p Piece) GenerateMoves(game *Game) []PlainMove {
	switch p.Type {
	case Pawn:
		return GeneratePawnMoves(p, game)
	case Bishop:
		return GenerateBishopMoves(p, game)
	case Rook:
		return GenerateRookMoves(p, game)
	case Knight:
		return GenerateKnightMoves(p, game)
	case Queen:
		return GenerateQueenMoves(p, game)
	case King:
		return GenerateKingMoves(p, game)
	default:
		return []PlainMove{}
	}
}


func (p Piece) GenerateMovesInto(game *Game, out []PlainMove) []PlainMove {
	switch p.Type {
	case Pawn:
		return GeneratePawnMovesInto(p, game, out)
	case Bishop:
		return GenerateBishopMovesInto(p, game, out)
	case Rook:
		return GenerateRookMovesInto(p, game, out)
	case Knight:
		return GenerateKnightMovesInto(p, game, out)
	case Queen:
		return GenerateQueenMovesInto(p, game, out)
	case King:
		return GenerateKingMovesInto(p, game, out)
	default:
		return out
	}
}

func (p Piece) AttacksSquare(game *Game, target Position) bool {
	switch p.Type {
	case Pawn:
		return p.pawnAttacksSquare(target)
	case Bishop:
		return p.bishopAttacksSquare(game, target)
	case Rook:
		return p.rookAttacksSquare(game, target)
	case Knight:
		return p.knightAttacksSquare(target)
	case Queen:
		return p.queenAttacksSquare(game, target)
	case King:
		return p.kingAttacksSquare(target)
	default:
		return false
	}
}

func (p Piece) pawnAttacksSquare(target Position) bool {
	var diagLeftDir, diagRightDir Direction
	if p.Color == Black {
		diagLeftDir = DirDownLeft
		diagRightDir = DirDownRight
	} else {
		diagLeftDir = DirUpLeft
		diagRightDir = DirUpRight
	}

	leftAttack := p.Position.AddReturn(diagLeftDir)
	rightAttack := p.Position.AddReturn(diagRightDir)

	return target.Equals(leftAttack) || target.Equals(rightAttack)
}

func (p Piece) bishopAttacksSquare(game *Game, target Position) bool {
	for _, dir := range bishopDirs {
		dirOne, dirTwo := dir[0], dir[1]
		for mag := 1; ; mag++ {
			pos := p.Position.AddVectorsReturn(Vector{dirOne, mag}, Vector{dirTwo, mag})
			if !pos.InBoard() {
				break
			}
			if pos.Equals(target) {
				return true
			}
			if GetPiece(&game.Board, pos) != nil {
				break
			}
		}
	}
	return false
}

func (p Piece) rookAttacksSquare(game *Game, target Position) bool {
	for _, dir := range rookDirs {
		for mag := 1; ; mag++ {
			pos := p.Position.AddVectorReturn(Vector{dir, mag})
			if !pos.InBoard() {
				break
			}
			if pos.Equals(target) {
				return true
			}
			if GetPiece(&game.Board, pos) != nil {
				break
			}
		}
	}
	return false
}

func (p Piece) knightAttacksSquare(target Position) bool {
	for _, offset := range knightOffsets {
		pos := p.Position.AddVectorsReturn(offset[0], offset[1])
		if pos.Equals(target) {
			return true
		}
	}
	return false
}

func (p Piece) queenAttacksSquare(game *Game, target Position) bool {
	return p.rookAttacksSquare(game, target) || p.bishopAttacksSquare(game, target)
}

func (p Piece) kingAttacksSquare(target Position) bool {
	for _, dirs := range kingOffsets {
		pos := p.Position
		for _, d := range dirs {
			pos = pos.AddReturn(d)
		}
		if pos.Equals(target) {
			return true
		}
	}
	return false
}

func IsEnemyPiece(game *Game, pos Position, piece Piece) bool {
	if GetPiece(&game.Board, pos) == nil {
		return false
	}

	return GetPiece(&game.Board, pos).Color != piece.Color
}

var promotionTypes = []PieceType{Queen, Rook, Bishop, Knight}

func GeneratePawnMoves(piece Piece, game *Game) []PlainMove {
	from := piece.Position
	var dir, diagLeftDir, diagRightDir Direction
	if piece.Color == Black {
		dir = DirDown
		diagLeftDir = DirDownLeft
		diagRightDir = DirDownRight
	} else {
		dir = DirUp
		diagLeftDir = DirUpLeft
		diagRightDir = DirUpRight
	}

	out := make([]PlainMove, 0, 8)

	oneForward := from.AddReturn(dir)

	if oneForward.InBoard() && GetPiece(&game.Board, oneForward) == nil {
		if IsPromotionSquare(oneForward, piece.Color) {
			for _, pt := range promotionTypes {
				pt := pt
				out = append(out, PlainMove{from, oneForward, &pt})
			}
		} else {
			out = append(out, PlainMove{from, oneForward, nil})

			if !piece.Moved {
				twoForward := oneForward.AddReturn(dir)
				if twoForward.InBoard() && GetPiece(&game.Board, twoForward) == nil {
					out = append(out, PlainMove{from, twoForward, nil})
				}
			}
		}
	}

	leftCapture := from.AddReturn(diagLeftDir)
	if leftCapture.InBoard() {
		if target := GetPiece(&game.Board, leftCapture); target != nil && target.Color != piece.Color {
			if IsPromotionSquare(leftCapture, piece.Color) {
				for _, pt := range promotionTypes {
					pt := pt
					out = append(out, PlainMove{from, leftCapture, &pt})
				}
			} else {
				out = append(out, PlainMove{from, leftCapture, nil})
			}
		}
	}

	rightCapture := from.AddReturn(diagRightDir)
	if rightCapture.InBoard() {
		if target := GetPiece(&game.Board, rightCapture); target != nil && target.Color != piece.Color {
			if IsPromotionSquare(rightCapture, piece.Color) {
				for _, pt := range promotionTypes {
					pt := pt
					out = append(out, PlainMove{from, rightCapture, &pt})
				}
			} else {
				out = append(out, PlainMove{from, rightCapture, nil})
			}
		}
	}

	if game.EPPosition != nil && *game.EPColor != piece.Color {
		if leftCapture.Equals(*game.EPPosition) {
			out = append(out, PlainMove{from, leftCapture, nil})
		} else if rightCapture.Equals(*game.EPPosition) {
			out = append(out, PlainMove{from, rightCapture, nil})
		}
	}

	return out
}

func GenerateBishopMoves(piece Piece, game *Game) []PlainMove {
	from := piece.Position
	out := make([]PlainMove, 0, 16)

	for _, dir := range bishopDirs {
		dirOne, dirTwo := dir[0], dir[1]
		for mag := 1; ; mag++ {
			pos := from.AddVectorsReturn(Vector{dirOne, mag}, Vector{dirTwo, mag})
			if !pos.InBoard() {
				break
			}
			if GetPiece(&game.Board, pos) != nil {
				if GetPiece(&game.Board, pos).Color != piece.Color {
					out = append(out, PlainMove{from, pos, nil})
				}
				break
			}
			out = append(out, PlainMove{from, pos, nil})
		}
	}

	return out
}

func GenerateRookMoves(piece Piece, game *Game) []PlainMove {
	from := piece.Position
	out := make([]PlainMove, 0, 16)

	for _, dir := range rookDirs {
		for mag := 1; ; mag++ {
			pos := from.AddVectorReturn(Vector{dir, mag})
			if !pos.InBoard() {
				break
			}
			if GetPiece(&game.Board, pos) != nil {
				if GetPiece(&game.Board, pos).Color != piece.Color {
					out = append(out, PlainMove{from, pos, nil})
				}
				break
			}
			out = append(out, PlainMove{from, pos, nil})
		}
	}

	return out
}

func GenerateKnightMoves(piece Piece, game *Game) []PlainMove {
	from := piece.Position
	out := make([]PlainMove, 0, 12)

	for _, offset := range knightOffsets {
		pos := from.AddVectorsReturn(offset[0], offset[1])
		if !pos.InBoard() {
			continue
		}
		if GetPiece(&game.Board, pos) != nil {
			if GetPiece(&game.Board, pos).Color != piece.Color {
				out = append(out, PlainMove{from, pos, nil})
			}
			continue
		}
		out = append(out, PlainMove{from, pos, nil})
	}

	return out
}

func GenerateQueenMoves(piece Piece, game *Game) []PlainMove {
	out := make([]PlainMove, 0, 32)
	out = append(out, GenerateRookMoves(piece, game)...)
	out = append(out, GenerateBishopMoves(piece, game)...)
	return out
}

func GenerateKingMoves(piece Piece, game *Game) []PlainMove {
	from := piece.Position
	out := make([]PlainMove, 0, 12)

	for _, dirs := range kingOffsets {
		pos := from
		for _, d := range dirs {
			pos = pos.AddReturn(d)
		}

		if !pos.InBoard() {
			continue
		}
		if GetPiece(&game.Board, pos) != nil {
			if GetPiece(&game.Board, pos).Color != piece.Color {
				out = append(out, PlainMove{from, pos, nil})
			}
			continue
		}
		out = append(out, PlainMove{from, pos, nil})
	}

	return out
}



func GeneratePawnMovesInto(piece Piece, game *Game, out []PlainMove) []PlainMove {
	from := piece.Position
	var dir, diagLeftDir, diagRightDir Direction
	if piece.Color == Black {
		dir = DirDown
		diagLeftDir = DirDownLeft
		diagRightDir = DirDownRight
	} else {
		dir = DirUp
		diagLeftDir = DirUpLeft
		diagRightDir = DirUpRight
	}

	oneForward := from.AddReturn(dir)

	if oneForward.InBoard() && GetPiece(&game.Board, oneForward) == nil {
		if IsPromotionSquare(oneForward, piece.Color) {
			for i := range promotionTypes {
				out = append(out, PlainMove{from, oneForward, &promotionTypes[i]})
			}
		} else {
			out = append(out, PlainMove{from, oneForward, nil})

			if !piece.Moved {
				twoForward := oneForward.AddReturn(dir)
				if twoForward.InBoard() && GetPiece(&game.Board, twoForward) == nil {
					out = append(out, PlainMove{from, twoForward, nil})
				}
			}
		}
	}

	leftCapture := from.AddReturn(diagLeftDir)
	if leftCapture.InBoard() {
		if target := GetPiece(&game.Board, leftCapture); target != nil && target.Color != piece.Color {
			if IsPromotionSquare(leftCapture, piece.Color) {
				for i := range promotionTypes {
					out = append(out, PlainMove{from, leftCapture, &promotionTypes[i]})
				}
			} else {
				out = append(out, PlainMove{from, leftCapture, nil})
			}
		}
	}

	rightCapture := from.AddReturn(diagRightDir)
	if rightCapture.InBoard() {
		if target := GetPiece(&game.Board, rightCapture); target != nil && target.Color != piece.Color {
			if IsPromotionSquare(rightCapture, piece.Color) {
				for i := range promotionTypes {
					out = append(out, PlainMove{from, rightCapture, &promotionTypes[i]})
				}
			} else {
				out = append(out, PlainMove{from, rightCapture, nil})
			}
		}
	}

	if game.EPPosition != nil && *game.EPColor != piece.Color {
		if leftCapture.Equals(*game.EPPosition) {
			out = append(out, PlainMove{from, leftCapture, nil})
		} else if rightCapture.Equals(*game.EPPosition) {
			out = append(out, PlainMove{from, rightCapture, nil})
		}
	}

	return out
}

func GenerateBishopMovesInto(piece Piece, game *Game, out []PlainMove) []PlainMove {
	from := piece.Position

	for _, dir := range bishopDirs {
		dirOne, dirTwo := dir[0], dir[1]
		for mag := 1; ; mag++ {
			pos := from.AddVectorsReturn(Vector{dirOne, mag}, Vector{dirTwo, mag})
			if !pos.InBoard() {
				break
			}
			if p := GetPiece(&game.Board, pos); p != nil {
				if p.Color != piece.Color {
					out = append(out, PlainMove{from, pos, nil})
				}
				break
			}
			out = append(out, PlainMove{from, pos, nil})
		}
	}

	return out
}

func GenerateRookMovesInto(piece Piece, game *Game, out []PlainMove) []PlainMove {
	from := piece.Position

	for _, dir := range rookDirs {
		for mag := 1; ; mag++ {
			pos := from.AddVectorReturn(Vector{dir, mag})
			if !pos.InBoard() {
				break
			}
			if p := GetPiece(&game.Board, pos); p != nil {
				if p.Color != piece.Color {
					out = append(out, PlainMove{from, pos, nil})
				}
				break
			}
			out = append(out, PlainMove{from, pos, nil})
		}
	}

	return out
}

func GenerateKnightMovesInto(piece Piece, game *Game, out []PlainMove) []PlainMove {
	from := piece.Position

	for _, offset := range knightOffsets {
		pos := from.AddVectorsReturn(offset[0], offset[1])
		if !pos.InBoard() {
			continue
		}
		if p := GetPiece(&game.Board, pos); p != nil {
			if p.Color != piece.Color {
				out = append(out, PlainMove{from, pos, nil})
			}
			continue
		}
		out = append(out, PlainMove{from, pos, nil})
	}

	return out
}

func GenerateQueenMovesInto(piece Piece, game *Game, out []PlainMove) []PlainMove {
	out = GenerateRookMovesInto(piece, game, out)
	out = GenerateBishopMovesInto(piece, game, out)
	return out
}

func GenerateKingMovesInto(piece Piece, game *Game, out []PlainMove) []PlainMove {
	from := piece.Position

	for _, dirs := range kingOffsets {
		pos := from
		for _, d := range dirs {
			pos = pos.AddReturn(d)
		}

		if !pos.InBoard() {
			continue
		}
		if p := GetPiece(&game.Board, pos); p != nil {
			if p.Color != piece.Color {
				out = append(out, PlainMove{from, pos, nil})
			}
			continue
		}
		out = append(out, PlainMove{from, pos, nil})
	}

	return out
}

func (p Piece) Draw(screen *ebiten.Image, game *Game) error {
	img, err := LoadImage(p.Image)
	if err != nil {
		return err
	}

	op := &ebiten.DrawImageOptions{}
	scale := float64(game.PieceScale) / 100
	op.GeoM.Scale(scale, scale)

	w, h := img.Size()
	x, y := p.Position.PositionToCoordinate(float64(game.HexagonSize), float64(game.ScreenWidth/2), float64(game.ScreenHeight/2))
	op.GeoM.Translate(x-float64(w)*scale/2, y-float64(h)*scale/2)
	screen.DrawImage(img, op)

	return nil
}

func DrawPieceAtCoord(screen *ebiten.Image, imagePath string, x, y, scale float64) error {
	img, err := LoadImage(imagePath)
	if err != nil {
		return err
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)

	w, h := img.Size()
	op.GeoM.Translate(x-float64(w)*scale/2, y-float64(h)*scale/2)
	screen.DrawImage(img, op)

	return nil
}
