package internal

var pieceValues = map[PieceType]int{
	Pawn:   100,
	Knight: 300,
	Bishop: 350,
	Rook:   500,
	Queen:  900,
	King:   0,
}

const (
	passedPawnBonus        = 20 // Base bonus for passed pawn
	passedPawnAdvanceBonus = 5  // Additional bonus per rank advanced
	connectedPassedBonus   = 15 // Bonus for connected passed pawns
	doubledPawnPenalty     = 15 // Penalty for doubled pawns
	isolatedPawnPenalty    = 10 // Penalty for isolated pawns
)

const (
	pawnShieldBonus     = 10 // Bonus per pawn shielding the king
	kingAttackerPenalty = 8  // Penalty per enemy piece attacking king zone
	openFileNearKingPen = 15 // Penalty for open file adjacent to king
	kingExposedPenalty  = 20 // Penalty if king has few nearby defenders
)

const (
	knightMobilityBonus = 4 // Knights benefit most from mobility
	bishopMobilityBonus = 3 // Bishops are long-range
	rookMobilityBonus   = 2 // Rooks on open files
	queenMobilityBonus  = 1 // Queen mobility less important (already powerful)
)

func PieceValue(piece *Piece) int {
	return pieceValues[piece.Type]
}

func centerDistance(pos Position) int {
	absX := pos.X
	if absX < 0 {
		absX = -absX
	}
	absY := pos.Y
	if absY < 0 {
		absY = -absY
	}
	absZ := pos.Z
	if absZ < 0 {
		absZ = -absZ
	}

	max := absX
	if absY > max {
		max = absY
	}
	if absZ > max {
		max = absZ
	}
	return max
}

func getAdjacentFileDirections(color Color) (Direction, Direction) {
	if color == White {
		return DirUpLeft, DirUpRight
	}
	return DirDownLeft, DirDownRight
}

func isPassedPawn(pawn *Piece, game *Game) bool {
	pos := pawn.Position
	var forwardDir Direction
	var adjLeft, adjRight Direction

	if pawn.Color == White {
		forwardDir = DirUp
		adjLeft = DirUpLeft
		adjRight = DirUpRight
	} else {
		forwardDir = DirDown
		adjLeft = DirDownLeft
		adjRight = DirDownRight
	}

	enemyColor := pawn.Color.Other()

	for checkPos := pos.AddReturn(forwardDir); checkPos.InBoard(); checkPos = checkPos.AddReturn(forwardDir) {
		if piece := GetPiece(&game.Board, checkPos); piece != nil && piece.Type == Pawn && piece.Color == enemyColor {
			return false
		}
		leftPos := checkPos.AddReturn(adjLeft)
		if leftPos.InBoard() {
			if piece := GetPiece(&game.Board, leftPos); piece != nil && piece.Type == Pawn && piece.Color == enemyColor {
				return false
			}
		}
		rightPos := checkPos.AddReturn(adjRight)
		if rightPos.InBoard() {
			if piece := GetPiece(&game.Board, rightPos); piece != nil && piece.Type == Pawn && piece.Color == enemyColor {
				return false
			}
		}
	}

	return true
}

func isDoubledPawn(pawn *Piece, game *Game) bool {
	pos := pawn.Position

	for checkPos := pos.AddReturn(DirUp); checkPos.InBoard(); checkPos = checkPos.AddReturn(DirUp) {
		if piece := GetPiece(&game.Board, checkPos); piece != nil && piece.Type == Pawn && piece.Color == pawn.Color {
			return true
		}
	}
	for checkPos := pos.AddReturn(DirDown); checkPos.InBoard(); checkPos = checkPos.AddReturn(DirDown) {
		if piece := GetPiece(&game.Board, checkPos); piece != nil && piece.Type == Pawn && piece.Color == pawn.Color {
			return true
		}
	}

	return false
}

func isIsolatedPawn(pawn *Piece, game *Game) bool {
	pos := pawn.Position

	adjDirs := []Direction{DirUpLeft, DirUpRight, DirDownLeft, DirDownRight}

	for _, adjDir := range adjDirs {
		for checkPos := pos.AddReturn(adjDir); checkPos.InBoard(); checkPos = checkPos.AddReturn(adjDir) {
			if piece := GetPiece(&game.Board, checkPos); piece != nil && piece.Type == Pawn && piece.Color == pawn.Color {
				return false
			}
		}
	}

	return true
}

func hasConnectedPassedPawn(pawn *Piece, game *Game, passedPawns map[*Piece]bool) bool {
	pos := pawn.Position

	adjDirs := []Direction{DirUpLeft, DirUpRight, DirDownLeft, DirDownRight}

	for _, dir := range adjDirs {
		adjPos := pos.AddReturn(dir)
		if adjPos.InBoard() {
			if piece := GetPiece(&game.Board, adjPos); piece != nil && piece.Type == Pawn && piece.Color == pawn.Color {
				if passedPawns[piece] {
					return true
				}
			}
		}
	}

	return false
}

func getPawnAdvancement(pawn *Piece) int {
	if pawn.Color == White {
		return pawn.Position.Y + 1 // Ranges roughly from 0 to 6
	}
	return -pawn.Position.Y + 1 // Ranges roughly from 0 to 6
}

func evaluatePawnStructure(game *Game) int {
	score := 0

	passedPawns := make(map[*Piece]bool)

	for _, piece := range game.Board {
		if piece == nil || piece.Type != Pawn {
			continue
		}
		if isPassedPawn(piece, game) {
			passedPawns[piece] = true
		}
	}

	for _, piece := range game.Board {
		if piece == nil || piece.Type != Pawn {
			continue
		}

		pawnScore := 0

		if passedPawns[piece] {
			advancement := getPawnAdvancement(piece)
			pawnScore += passedPawnBonus + (advancement * passedPawnAdvanceBonus)

			if hasConnectedPassedPawn(piece, game, passedPawns) {
				pawnScore += connectedPassedBonus
			}
		}

		if isDoubledPawn(piece, game) {
			pawnScore -= doubledPawnPenalty
		}

		if isIsolatedPawn(piece, game) {
			pawnScore -= isolatedPawnPenalty
		}

		if piece.Color == White {
			score += pawnScore
		} else {
			score -= pawnScore
		}
	}

	return score
}

func getKingZone(kingPos Position) []Position {
	zone := make([]Position, 0, 13)
	zone = append(zone, kingPos)

	allDirs := [][]Direction{
		{DirUp},
		{DirDown},
		{DirUpLeft},
		{DirUpRight},
		{DirDownLeft},
		{DirDownRight},
		{DirUp, DirUpRight},
		{DirDownRight, DirUpRight},
		{DirDownRight, DirDown},
		{DirDownLeft, DirDown},
		{DirDownLeft, DirUpLeft},
		{DirUp, DirUpLeft},
	}

	for _, dirs := range allDirs {
		pos := kingPos
		for _, d := range dirs {
			pos = pos.AddReturn(d)
		}
		if pos.InBoard() {
			zone = append(zone, pos)
		}
	}

	return zone
}

func countPawnShield(kingPos Position, color Color, game *Game) int {
	count := 0

	var shieldDirs []Direction
	if color == White {
		shieldDirs = []Direction{DirUp, DirUpLeft, DirUpRight}
	} else {
		shieldDirs = []Direction{DirDown, DirDownLeft, DirDownRight}
	}

	for _, dir := range shieldDirs {
		for dist := 1; dist <= 2; dist++ {
			pos := kingPos
			for i := 0; i < dist; i++ {
				pos = pos.AddReturn(dir)
			}
			if pos.InBoard() {
				if piece := GetPiece(&game.Board, pos); piece != nil && piece.Type == Pawn && piece.Color == color {
					count++
				}
			}
		}
	}

	return count
}

func countKingZoneAttackers(kingPos Position, enemyColor Color, game *Game) int {
	attackers := 0

	for _, piece := range game.Board {
		if piece == nil || piece.Color != enemyColor {
			continue
		}
		if piece.Type == Pawn || piece.Type == King {
			continue
		}

		dist := hexDistance(piece.Position, kingPos)

		if piece.Type == Queen && dist <= 4 {
			attackers += 2 // Queen is more dangerous
		} else if (piece.Type == Rook || piece.Type == Bishop) && dist <= 3 {
			attackers++
		} else if piece.Type == Knight && dist <= 3 {
			attackers++
		}
	}

	return attackers
}

func hexDistance(a, b Position) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dz < 0 {
		dz = -dz
	}
	max := dx
	if dy > max {
		max = dy
	}
	if dz > max {
		max = dz
	}
	return max
}

func countDefenders(kingZone []Position, color Color, game *Game) int {
	defenders := 0

	for _, pos := range kingZone {
		if piece := GetPiece(&game.Board, pos); piece != nil && piece.Color == color && piece.Type != King {
			defenders++
		}
	}

	return defenders
}

func hasOpenFileNearKing(kingPos Position, color Color, game *Game) bool {

	filesToCheck := []Position{kingPos}

	adjDirs := []Direction{DirUpLeft, DirUpRight, DirDownLeft, DirDownRight}
	for _, dir := range adjDirs {
		adjPos := kingPos.AddReturn(dir)
		if adjPos.InBoard() {
			filesToCheck = append(filesToCheck, adjPos)
		}
	}

	for _, filePos := range filesToCheck {
		hasPawn := false
		for checkPos := filePos; checkPos.InBoard(); checkPos = checkPos.AddReturn(DirUp) {
			if piece := GetPiece(&game.Board, checkPos); piece != nil && piece.Type == Pawn {
				hasPawn = true
				break
			}
		}
		if !hasPawn {
			for checkPos := filePos.AddReturn(DirDown); checkPos.InBoard(); checkPos = checkPos.AddReturn(DirDown) {
				if piece := GetPiece(&game.Board, checkPos); piece != nil && piece.Type == Pawn {
					hasPawn = true
					break
				}
			}
		}
		if !hasPawn {
			return true // Found an open file
		}
	}

	return false
}

func evaluateMobility(game *Game) int {
	score := 0

	for _, piece := range game.Board {
		if piece == nil {
			continue
		}

		var mobilityBonus int
		switch piece.Type {
		case Knight:
			dist := centerDistance(piece.Position)
			if dist >= 4 {
				mobilityBonus = -15 // Knight on rim is dim
			} else if dist >= 3 {
				mobilityBonus = -5
			}

		case Bishop:
			blocked := 0
			for _, dir := range bishopDirs {
				pos := piece.Position.AddVectorsReturn(Vector{dir[0], 1}, Vector{dir[1], 1})
				if pos.InBoard() {
					if blocker := GetPiece(&game.Board, pos); blocker != nil && blocker.Type == Pawn && blocker.Color == piece.Color {
						blocked++
					}
				}
			}
			mobilityBonus = -blocked * 5 // Penalty for each blocking pawn

		case Rook:
			hasOpenFile := true
			for checkPos := piece.Position.AddReturn(DirUp); checkPos.InBoard(); checkPos = checkPos.AddReturn(DirUp) {
				if p := GetPiece(&game.Board, checkPos); p != nil && p.Type == Pawn && p.Color == piece.Color {
					hasOpenFile = false
					break
				}
			}
			if hasOpenFile {
				for checkPos := piece.Position.AddReturn(DirDown); checkPos.InBoard(); checkPos = checkPos.AddReturn(DirDown) {
					if p := GetPiece(&game.Board, checkPos); p != nil && p.Type == Pawn && p.Color == piece.Color {
						hasOpenFile = false
						break
					}
				}
			}
			if hasOpenFile {
				mobilityBonus = 15
			}

		default:
			continue
		}

		if piece.Color == White {
			score += mobilityBonus
		} else {
			score -= mobilityBonus
		}
	}

	return score
}

func evaluateKingSafety(game *Game) int {
	score := 0

	whiteKingZone := getKingZone(game.WhiteKing)
	whitePawnShield := countPawnShield(game.WhiteKing, White, game)
	whiteAttackers := countKingZoneAttackers(game.WhiteKing, Black, game)
	whiteDefenders := countDefenders(whiteKingZone, White, game)

	whiteKingSafety := whitePawnShield * pawnShieldBonus
	whiteKingSafety -= whiteAttackers * kingAttackerPenalty
	if whiteDefenders < 2 {
		whiteKingSafety -= kingExposedPenalty
	}
	if hasOpenFileNearKing(game.WhiteKing, White, game) {
		whiteKingSafety -= openFileNearKingPen
	}

	blackKingZone := getKingZone(game.BlackKing)
	blackPawnShield := countPawnShield(game.BlackKing, Black, game)
	blackAttackers := countKingZoneAttackers(game.BlackKing, White, game)
	blackDefenders := countDefenders(blackKingZone, Black, game)

	blackKingSafety := blackPawnShield * pawnShieldBonus
	blackKingSafety -= blackAttackers * kingAttackerPenalty
	if blackDefenders < 2 {
		blackKingSafety -= kingExposedPenalty
	}
	if hasOpenFileNearKing(game.BlackKing, Black, game) {
		blackKingSafety -= openFileNearKingPen
	}

	score = whiteKingSafety - blackKingSafety
	return score
}

func Evaluate(game *Game) int {
	score := 0

	for _, piece := range game.Board {
		if piece == nil {
			continue
		}

		value := pieceValues[piece.Type]

		centerBonus := (5 - centerDistance(piece.Position)) * 2

		if piece.Type == Knight || piece.Type == Bishop {
			centerBonus *= 2
		}

		if piece.Type == Pawn {
			if piece.Color == White {
				advancement := 5 + piece.Position.Y
				if advancement > 0 {
					value += advancement * 5
				}
			} else {
				advancement := 5 - piece.Position.Y
				if advancement > 0 {
					value += advancement * 5
				}
			}
		}

		if (piece.Type == Knight || piece.Type == Bishop) && piece.Moved {
			value += 40
		}

		totalValue := value + centerBonus

		if piece.Color == White {
			score += totalValue
		} else {
			score -= totalValue
		}
	}

	score += evaluatePawnStructure(game)

	score += evaluateKingSafety(game)

	score += evaluateMobility(game)

	return score
}

func EvaluateForSide(game *Game, color Color) int {
	score := Evaluate(game)
	if color == Black {
		return -score
	}
	return score
}

// Global NNUE model - loaded once at startup
var nnueModel *Model

func init() {
	model := LoadModel("models/prob-512-128-32-d4.json")
	nnueModel = &model
}

// EvaluateNNUE returns evaluation in centipawns using the NNUE model
// Positive = White advantage, Negative = Black advantage
func EvaluateNNUE(game *Game) int {
	stm, nstm := BoardToHalfKPFeaturesAlloc(game)
	winProb := nnueModel.Predict(stm, nstm)

	// Convert probability to centipawns
	// 0.5 = 0cp, 1.0 = +1000cp, 0.0 = -1000cp (approximate)
	// Using a sigmoid-inverse-like scaling
	cp := int((winProb - 0.5) * 2000)

	// From side-to-move perspective, convert to White's perspective
	if game.Turn == Black {
		cp = -cp
	}

	return cp
}

// EvaluateNNUEForSide returns NNUE evaluation from the given color's perspective
func EvaluateNNUEForSide(game *Game, color Color) int {
	score := EvaluateNNUE(game)
	if color == Black {
		return -score
	}
	return score
}

// GetNNUEModel returns the global NNUE model for direct access (e.g., incremental updates)
func GetNNUEModel() *Model {
	return nnueModel
}
