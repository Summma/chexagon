package internal

import (
	"math/rand"
)

var (
	zobristPieces [6][2][91]uint64

	zobristSideToMove uint64

	zobristEnPassant [91]uint64

	positionToIndex map[Position]int
	indexToPosition []Position
)

func init() {
	r := rand.New(rand.NewSource(0x48455843484553))

	positionToIndex = make(map[Position]int)
	indexToPosition = make([]Position, 0, 91)

	idx := 0
	for q := -5; q <= 5; q++ {
		for r := -5; r <= 5; r++ {
			s := -q - r
			if s >= -5 && s <= 5 {
				pos := Position{q, r, s}
				if pos.InBoard() {
					positionToIndex[pos] = idx
					indexToPosition = append(indexToPosition, pos)
					idx++
				}
			}
		}
	}

	for pt := 0; pt < 6; pt++ {
		for c := 0; c < 2; c++ {
			for i := 0; i < 91; i++ {
				zobristPieces[pt][c][i] = r.Uint64()
			}
		}
	}

	zobristSideToMove = r.Uint64()

	for i := 0; i < 91; i++ {
		zobristEnPassant[i] = r.Uint64()
	}

	InitPosToIdxTable()
}

func (g *Game) ComputeHash() uint64 {
	var hash uint64

	for idx, piece := range g.Board {
		if piece != nil {
			hash ^= zobristPieces[piece.Type][piece.Color][idx]
		}
	}

	if g.Turn == Black {
		hash ^= zobristSideToMove
	}

	if g.EPPosition != nil {
		idx := positionToIndex[*g.EPPosition]
		hash ^= zobristEnPassant[idx]
	}

	return hash
}

func UpdateHashForMove(hash uint64, move Move, undo MoveUndo, game *Game) uint64 {
	fromIdx := positionToIndex[move.From]
	toIdx := positionToIndex[move.To]

	hash ^= zobristPieces[move.PieceType][move.PieceColor][fromIdx]

	destType := move.PieceType
	if move.IsPromotion && move.PromotionType != nil {
		destType = *move.PromotionType
	}
	hash ^= zobristPieces[destType][move.PieceColor][toIdx]

	if undo.CapturedPiece != nil {
		capIdx := positionToIndex[undo.CapturedPos]
		hash ^= zobristPieces[undo.CapturedPiece.Type][undo.CapturedPiece.Color][capIdx]
	}

	hash ^= zobristSideToMove

	if undo.EPPosBefore != nil {
		idx := positionToIndex[*undo.EPPosBefore]
		hash ^= zobristEnPassant[idx]
	}
	if game.EPPosition != nil {
		idx := positionToIndex[*game.EPPosition]
		hash ^= zobristEnPassant[idx]
	}

	return hash
}

func UpdateHashForUnmove(hash uint64, move Move, undo MoveUndo, game *Game) uint64 {
	fromIdx := positionToIndex[move.From]
	toIdx := positionToIndex[move.To]

	destType := move.PieceType
	if move.IsPromotion && move.PromotionType != nil {
		destType = *move.PromotionType
	}
	hash ^= zobristPieces[destType][move.PieceColor][toIdx]

	hash ^= zobristPieces[move.PieceType][move.PieceColor][fromIdx]

	if undo.CapturedPiece != nil {
		capIdx := positionToIndex[undo.CapturedPos]
		hash ^= zobristPieces[undo.CapturedPiece.Type][undo.CapturedPiece.Color][capIdx]
	}

	hash ^= zobristSideToMove

	if game.EPPosition != nil {
		idx := positionToIndex[*game.EPPosition]
		hash ^= zobristEnPassant[idx]
	}
	if undo.EPPosBefore != nil {
		idx := positionToIndex[*undo.EPPosBefore]
		hash ^= zobristEnPassant[idx]
	}

	return hash
}
