package internal

import (
	"fmt"
	"sync"
)

var moveListPool = sync.Pool{
	New: func() interface{} {
		moves := make([]Move, 0, 128)
		return &moves
	},
}

func GetMoveList() *[]Move {
	moves := moveListPool.Get().(*[]Move)
	*moves = (*moves)[:0]
	return moves
}

func PutMoveList(moves *[]Move) {
	if moves != nil {
		moveListPool.Put(moves)
	}
}

type PlainMove struct {
	From          Position
	To            Position
	PromotionType *PieceType
}

type Move struct {
	From          Position
	To            Position
	PieceType     PieceType
	PieceColor    Color
	ToPiece       *Piece
	IsValid       bool
	IsCapture     bool
	IsCheck       bool
	IsCheckmate   bool
	IsEnPassant   bool
	IsPromotion   bool
	IsDoublePawn  bool
	IsRepetition  bool
	PromotionType *PieceType
}

func (m Move) String() string {
	return fmt.Sprintf("Move{%s %s to %v, valid=%t, capture=%t, check=%t, checkmate=%t, ep=%t, promo=%t}",
		m.PieceColor, m.PieceType, m.To,
		m.IsValid, m.IsCapture, m.IsCheck, m.IsCheckmate,
		m.IsEnPassant, m.IsPromotion,
	)
}

type MoveUndo struct {
	CapturedPiece    *Piece
	CapturedPos      Position
	EPPosBefore      *Position
	EPColorBefore    *Color
	PieceMovedBefore bool
	KingPosBefore    Position
	PieceTypeBefore  PieceType
	PieceImageBefore string
	HashBefore       uint64
}

func (g *Game) MakeMove(move Move) MoveUndo {
	piece := GetPiece(&g.Board, move.From)
	undo := MoveUndo{
		CapturedPiece:    GetPiece(&g.Board, move.To),
		CapturedPos:      move.To,
		EPPosBefore:      g.EPPosition,
		EPColorBefore:    g.EPColor,
		PieceMovedBefore: piece.Moved,
		PieceTypeBefore:  piece.Type,
		PieceImageBefore: piece.Image,
		HashBefore:       g.Hash,
	}

	SetPiece(&g.Board, move.From, nil)
	piece.Position = move.To
	SetPiece(&g.Board, move.To, piece)
	piece.Moved = true

	if move.IsPromotion && move.PromotionType != nil {
		piece.Type = *move.PromotionType
		colorStr := "white"
		if piece.Color == Black {
			colorStr = "black"
		}
		typeStr := map[PieceType]string{
			Knight: "knight",
			Bishop: "bishop",
			Rook:   "rook",
			Queen:  "queen",
		}[*move.PromotionType]
		piece.Image = "assets/pieces-basic-png/" + colorStr + "-" + typeStr + ".png"
	}

	if move.IsEnPassant && move.ToPiece != nil {
		undo.CapturedPiece = move.ToPiece
		undo.CapturedPos = move.ToPiece.Position
		SetPiece(&g.Board, move.ToPiece.Position, nil)
	}

	if move.PieceType == King {
		if piece.Color == White {
			undo.KingPosBefore = g.WhiteKing
			g.WhiteKing = move.To
		} else {
			undo.KingPosBefore = g.BlackKing
			g.BlackKing = move.To
		}
	}

	g.EPPosition = nil
	g.EPColor = nil
	if move.PieceType == Pawn {
		var dir Direction
		if piece.Color == White {
			dir = DirUp
		} else {
			dir = DirDown
		}
		twoForward := move.From.AddVectorReturn(Vector{dir, 2})
		if move.To.Equals(twoForward) {
			epPos := move.From.AddReturn(dir)
			g.EPPosition = &epPos
			g.EPColor = &piece.Color
		}
	}

	g.Hash = UpdateHashForMove(undo.HashBefore, move, undo, g)

	g.PositionHistory[g.Hash]++

	return undo
}

func (g *Game) UnmakeMove(move Move, undo MoveUndo) {
	g.PositionHistory[g.Hash]--
	if g.PositionHistory[g.Hash] == 0 {
		delete(g.PositionHistory, g.Hash)
	}

	piece := GetPiece(&g.Board, move.To)
	SetPiece(&g.Board, move.To, nil)
	piece.Position = move.From
	SetPiece(&g.Board, move.From, piece)
	piece.Moved = undo.PieceMovedBefore
	piece.Type = undo.PieceTypeBefore
	piece.Image = undo.PieceImageBefore

	if undo.CapturedPiece != nil {
		SetPiece(&g.Board, undo.CapturedPos, undo.CapturedPiece)
	}

	if undo.PieceTypeBefore == King {
		if piece.Color == White {
			g.WhiteKing = undo.KingPosBefore
		} else {
			g.BlackKing = undo.KingPosBefore
		}
	}

	g.EPPosition = undo.EPPosBefore
	g.EPColor = undo.EPColorBefore

	g.Hash = undo.HashBefore
}

func GenerateAllLegalMoves(game *Game) []Move {
	return GenerateAllLegalMovesInto(game, nil)
}

func GenerateAllLegalMovesInto(game *Game, out []Move) []Move {
	if out == nil {
		out = make([]Move, 0, 96)
	} else {
		out = out[:0]
	}

	for _, piece := range game.Board {
		if piece == nil || piece.Color != game.Turn {
			continue
		}

		plainMoves := piece.GenerateMoves(game)
		for _, pm := range plainMoves {
			move := Move{
				From:          pm.From,
				To:            pm.To,
				PieceType:     piece.Type,
				PieceColor:    piece.Color,
				PromotionType: pm.PromotionType,
				IsValid:       true,
			}

			if GetPiece(&game.Board, pm.To) != nil {
				move.IsCapture = true
				move.ToPiece = GetPiece(&game.Board, pm.To)
			}

			if piece.Type == Pawn && game.EPPosition != nil && pm.To.Equals(*game.EPPosition) {
				move.IsEnPassant = true
				move.IsCapture = true
				if piece.Color == White {
					move.ToPiece = GetPiece(&game.Board, pm.To.AddReturn(DirDown))
				} else {
					move.ToPiece = GetPiece(&game.Board, pm.To.AddReturn(DirUp))
				}
			}

			if piece.Type == Pawn && IsPromotionSquare(pm.To, piece.Color) {
				move.IsPromotion = true
			}

			undo := game.MakeMove(move)
			legal := !IsKingInCheck(game, piece.Color)

			if legal && game.PositionHistory[game.Hash] >= 3 {
				move.IsRepetition = true
			}

			game.UnmakeMove(move, undo)

			if legal {
				out = append(out, move)
			}
		}
	}

	return out
}

func IsSquareAttacked(game *Game, pos Position, byColor Color) bool {
	for _, piece := range game.Board {
		if piece == nil || piece.Color != byColor {
			continue
		}
		if piece.AttacksSquare(game, pos) {
			return true
		}
	}
	return false
}

func IsKingInCheck(game *Game, color Color) bool {
	kingPos := game.WhiteKing
	if color == Black {
		kingPos = game.BlackKing
	}
	return IsSquareAttacked(game, kingPos, color.Other())
}

func HasLegalMoves(game *Game, color Color) bool {
	for _, piece := range game.Board {
		if piece == nil || piece.Color != color {
			continue
		}

		moves := piece.GenerateMoves(game)

		for _, plainMove := range moves {
			move := Move{
				IsValid:    true,
				From:       plainMove.From,
				To:         plainMove.To,
				PieceType:  piece.Type,
				PieceColor: piece.Color,
			}

			if GetPiece(&game.Board, plainMove.To) != nil && GetPiece(&game.Board, plainMove.To).Color != color {
				move.IsCapture = true
				move.ToPiece = GetPiece(&game.Board, plainMove.To)
			}

			undo := game.MakeMove(move)
			inCheck := IsKingInCheck(game, color)
			game.UnmakeMove(move, undo)

			if !inCheck {
				return true
			}
		}
	}
	return false
}

func IsCheckmate(game *Game, color Color) bool {
	return IsKingInCheck(game, color) && !HasLegalMoves(game, color)
}

func IsPromotionSquare(pos Position, color Color) bool {
	if color == White {
		nextPos := pos.AddReturn(DirUp)
		return !nextPos.InBoard()
	} else {
		nextPos := pos.AddReturn(DirDown)
		return !nextPos.InBoard()
	}
}

func IsSemiLegal(color Color, to Position, game *Game) (isSemiLegal, capture bool) {
	otherPiece := GetPiece(&game.Board, to)

	var obstructed bool
	if otherPiece != nil {
		if color != otherPiece.Color {
			capture = true
		} else {
			obstructed = true
		}
	}

	isSemiLegal = to.InBoard() && !obstructed

	return isSemiLegal, capture
}

func GetMove(from Position, to Position, game *Game) Move {
	return GetMoveWithPromotion(from, to, nil, game)
}

func GetMoveWithPromotion(from Position, to Position, promotionType *PieceType, game *Game) Move {
	piece := GetPiece(&game.Board, from)
	if piece == nil {
		return Move{IsValid: false}
	}

	move := Move{
		IsValid:       true,
		From:          from,
		To:            to,
		PieceType:     piece.Type,
		PieceColor:    piece.Color,
		PromotionType: promotionType,
	}

	if !to.InBoard() {
		move.IsValid = false
		return move
	}

	otherPiece := GetPiece(&game.Board, to)
	if otherPiece != nil {
		if piece.Color != otherPiece.Color {
			move.IsCapture = true
			move.ToPiece = otherPiece
		} else {
			move.IsValid = false
			return move
		}
	}

	switch piece.Type {
	case Pawn:
		move.IsValid = validatePawnMove(piece, from, to, game, &move)
	case Knight:
		move.IsValid = validateKnightMove(from, to)
	case Bishop:
		move.IsValid = validateBishopMove(from, to, game)
	case Rook:
		move.IsValid = validateRookMove(from, to, game)
	case Queen:
		move.IsValid = validateRookMove(from, to, game) || validateBishopMove(from, to, game)
	case King:
		move.IsValid = validateKingMove(from, to)
	}

	if !move.IsValid {
		return move
	}

	undo := game.MakeMove(move)

	if IsKingInCheck(game, piece.Color) {
		move.IsValid = false
	}

	if move.IsValid && IsKingInCheck(game, piece.Color.Other()) {
		move.IsCheck = true
		if IsCheckmate(game, piece.Color.Other()) {
			move.IsCheckmate = true
		}
	}

	game.UnmakeMove(move, undo)

	return move
}

func validatePawnMove(piece *Piece, from, to Position, game *Game, move *Move) bool {
	var dir, diagLeft, diagRight Direction
	if piece.Color == White {
		dir = DirUp
		diagLeft = DirUpLeft
		diagRight = DirUpRight
	} else {
		dir = DirDown
		diagLeft = DirDownLeft
		diagRight = DirDownRight
	}

	oneForward := from.AddReturn(dir)
	twoForward := oneForward.AddReturn(dir)
	leftCapture := from.AddReturn(diagLeft)
	rightCapture := from.AddReturn(diagRight)

	if IsPromotionSquare(to, piece.Color) {
		move.IsPromotion = true
	}

	if to.Equals(oneForward) {
		return GetPiece(&game.Board, to) == nil
	}

	if to.Equals(twoForward) && !piece.Moved {
		return GetPiece(&game.Board, oneForward) == nil && GetPiece(&game.Board, to) == nil
	}

	if to.Equals(leftCapture) || to.Equals(rightCapture) {
		if GetPiece(&game.Board, to) != nil && GetPiece(&game.Board, to).Color != piece.Color {
			return true
		}
		if game.EPPosition != nil && to.Equals(*game.EPPosition) && *game.EPColor != piece.Color {
			if piece.Color == White {
				move.ToPiece = GetPiece(&game.Board, to.AddReturn(DirDown))
			} else {
				move.ToPiece = GetPiece(&game.Board, to.AddReturn(DirUp))
			}
			move.IsEnPassant = true
			move.IsCapture = true
			return true
		}
	}

	return false
}

func validateKnightMove(from, to Position) bool {
	knightOffsets := [][]Vector{
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

	for _, offset := range knightOffsets {
		if from.AddVectorsReturn(offset...).Equals(to) {
			return true
		}
	}
	return false
}

func validateBishopMove(from, to Position, game *Game) bool {
	dirs := [][]Direction{
		{DirUp, DirUpRight},
		{DirDownRight, DirUpRight},
		{DirDownRight, DirDown},
		{DirDownLeft, DirDown},
		{DirDownLeft, DirUpLeft},
		{DirUp, DirUpLeft},
	}

	for _, dir := range dirs {
		dirOne, dirTwo := dir[0], dir[1]
		for mag := 1; ; mag++ {
			pos := from.AddVectorsReturn(Vector{dirOne, mag}, Vector{dirTwo, mag})
			if !pos.InBoard() {
				break
			}
			if pos.Equals(to) {
				return true
			}
			if GetPiece(&game.Board, pos) != nil {
				break
			}
		}
	}
	return false
}

func validateRookMove(from, to Position, game *Game) bool {
	dirs := []Direction{DirUp, DirDown, DirUpLeft, DirUpRight, DirDownLeft, DirDownRight}

	for _, dir := range dirs {
		for mag := 1; ; mag++ {
			pos := from.AddVectorReturn(Vector{dir, mag})
			if !pos.InBoard() {
				break
			}
			if pos.Equals(to) {
				return true
			}
			if GetPiece(&game.Board, pos) != nil {
				break
			}
		}
	}
	return false
}

func validateKingMove(from, to Position) bool {
	kingOffsets := [][]Direction{
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

	for _, dirs := range kingOffsets {
		pos := from
		for _, d := range dirs {
			pos = pos.AddReturn(d)
		}
		if pos.Equals(to) {
			return true
		}
	}
	return false
}
