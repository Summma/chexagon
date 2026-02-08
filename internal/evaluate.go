package internal


var pieceValues = [6]int{
	100,
	300,
	350,
	500,
	900,
	0,
}

const (
	passedPawnBonus        = 20
	passedPawnAdvanceBonus = 5
	connectedPassedBonus   = 15
	doubledPawnPenalty     = 15
	isolatedPawnPenalty    = 10
)

const (
	pawnShieldBonus     = 10
	kingAttackerPenalty = 8
	openFileNearKingPen = 15
	kingExposedPenalty  = 20
)

const (
	knightMobilityBonus = 4
	bishopMobilityBonus = 3
	rookMobilityBonus   = 2
	queenMobilityBonus  = 1
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
		return pawn.Position.Y + 1
	}
	return -pawn.Position.Y + 1
}


type pawnCacheEntry struct {
	hash  uint64
	score int
	valid bool
}

const pawnCacheSize = 1 << 16

var pawnCache [pawnCacheSize]pawnCacheEntry
var pawnZobrist [2][91]uint64


type kingSafetyCacheEntry struct {
	hash  uint64
	score int
	valid bool
}

const kingSafetyCacheSize = 1 << 16

var kingSafetyCache [kingSafetyCacheSize]kingSafetyCacheEntry

func init() {

	r := uint64(0x50415753545255)
	for c := 0; c < 2; c++ {
		for sq := 0; sq < 91; sq++ {
			r ^= r << 13
			r ^= r >> 7
			r ^= r << 17
			pawnZobrist[c][sq] = r
		}
	}

}


func (g *Game) ComputePawnHash() uint64 {
	var hash uint64
	for idx, piece := range g.Board {
		if piece != nil && piece.Type == Pawn {
			hash ^= pawnZobrist[piece.Color][idx]
		}
	}
	return hash
}

func evaluatePawnStructure(game *Game) int {

	pawnHash := game.PawnHash
	cacheIdx := pawnHash & (pawnCacheSize - 1)
	if entry := &pawnCache[cacheIdx]; entry.valid && entry.hash == pawnHash {
		return entry.score
	}
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


	pawnCache[cacheIdx] = pawnCacheEntry{hash: pawnHash, score: score, valid: true}

	return score
}


func getKingZoneIndices(kingIdx int) []int {
	return kingZones[kingIdx]
}

func countPawnShield(kingIdx int, color Color, game *Game) int {
	count := 0

	var shieldSquares []int
	if color == White {
		shieldSquares = pawnShieldSquaresWhite[kingIdx]
	} else {
		shieldSquares = pawnShieldSquaresBlack[kingIdx]
	}

	for _, sq := range shieldSquares {
		if piece := game.Board[sq]; piece != nil && piece.Type == Pawn && piece.Color == color {
			count++
		}
	}

	return count
}

func countKingZoneAttackers(kingPos Position, enemyColor Color, game *Game) int {
	attackers := 0
	kx, ky, kz := kingPos.X, kingPos.Y, kingPos.Z

	for _, piece := range game.Board {
		if piece == nil || piece.Color != enemyColor {
			continue
		}
		pt := piece.Type
		if pt == Pawn || pt == King {
			continue
		}


		dx := piece.Position.X - kx
		dy := piece.Position.Y - ky
		dz := piece.Position.Z - kz
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		if dz < 0 {
			dz = -dz
		}
		dist := dx
		if dy > dist {
			dist = dy
		}
		if dz > dist {
			dist = dz
		}

		if pt == Queen && dist <= 4 {
			attackers += 2
		} else if dist <= 3 {
			attackers++
		}
	}

	return attackers
}

func countDefenders(kingZoneIndices []int, color Color, game *Game) int {
	defenders := 0

	for _, idx := range kingZoneIndices {
		if piece := game.Board[idx]; piece != nil && piece.Color == color && piece.Type != King {
			defenders++
		}
	}

	return defenders
}

func hasOpenFileNearKing(kingPos Position, color Color, game *Game) bool {


	var filesToCheck [5]Position
	fileCount := 1
	filesToCheck[0] = kingPos


	adjDirs := [4]Direction{DirUpLeft, DirUpRight, DirDownLeft, DirDownRight}
	for _, dir := range adjDirs {
		adjPos := kingPos.AddReturn(dir)
		if adjPos.InBoard() {
			filesToCheck[fileCount] = adjPos
			fileCount++
		}
	}

	for i := 0; i < fileCount; i++ {
		filePos := filesToCheck[i]
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
			return true
		}
	}

	return false
}

func evaluateMobility(game *Game) int {
	score := 0

	for idx, piece := range game.Board {
		if piece == nil {
			continue
		}

		var mobilityBonus int
		switch piece.Type {
		case Knight:
			dist := centerDistance(piece.Position)
			if dist >= 4 {
				mobilityBonus = -15
			} else if dist >= 3 {
				mobilityBonus = -5
			}

		case Bishop:
			blocked := 0
			for _, adjIdx := range bishopAdjacent[idx] {
				if blocker := game.Board[adjIdx]; blocker != nil && blocker.Type == Pawn && blocker.Color == piece.Color {
					blocked++
				}
			}
			mobilityBonus = -blocked * 5

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

	cacheIdx := game.Hash & (kingSafetyCacheSize - 1)
	if entry := &kingSafetyCache[cacheIdx]; entry.valid && entry.hash == game.Hash {
		return entry.score
	}

	score := 0


	whiteKingIdx := fastPosToIdx(game.WhiteKing)
	blackKingIdx := fastPosToIdx(game.BlackKing)


	whiteKingZoneIndices := getKingZoneIndices(whiteKingIdx)
	whitePawnShield := countPawnShield(whiteKingIdx, White, game)
	whiteAttackers := countKingZoneAttackers(game.WhiteKing, Black, game)
	whiteDefenders := countDefenders(whiteKingZoneIndices, White, game)

	whiteKingSafety := whitePawnShield * pawnShieldBonus
	whiteKingSafety -= whiteAttackers * kingAttackerPenalty
	if whiteDefenders < 2 {
		whiteKingSafety -= kingExposedPenalty
	}
	if hasOpenFileNearKing(game.WhiteKing, White, game) {
		whiteKingSafety -= openFileNearKingPen
	}


	blackKingZoneIndices := getKingZoneIndices(blackKingIdx)
	blackPawnShield := countPawnShield(blackKingIdx, Black, game)
	blackAttackers := countKingZoneAttackers(game.BlackKing, White, game)
	blackDefenders := countDefenders(blackKingZoneIndices, Black, game)

	blackKingSafety := blackPawnShield * pawnShieldBonus
	blackKingSafety -= blackAttackers * kingAttackerPenalty
	if blackDefenders < 2 {
		blackKingSafety -= kingExposedPenalty
	}
	if hasOpenFileNearKing(game.BlackKing, Black, game) {
		blackKingSafety -= openFileNearKingPen
	}

	score = whiteKingSafety - blackKingSafety


	kingSafetyCache[cacheIdx] = kingSafetyCacheEntry{hash: game.Hash, score: score, valid: true}

	return score
}

func Evaluate(game *Game) int {
	score := 0

	for _, piece := range game.Board {
		if piece == nil {
			continue
		}

		pt := piece.Type
		value := pieceValues[pt]

		centerBonus := (5 - centerDistance(piece.Position)) * 2

		if pt == Knight || pt == Bishop {
			centerBonus *= 2
		}

		if pt == Pawn {
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

		if (pt == Knight || pt == Bishop) && piece.Moved {
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


var nnueModel *Model

func init() {
	model := LoadModel("models/prob-blended-d2.json")
	nnueModel = &model
}



func EvaluateNNUE(game *Game) int {
	stm, nstm := BoardToHalfKPFeaturesAlloc(game)
	winProb := nnueModel.Predict(stm, nstm)




	cp := int((winProb - 0.5) * 2000)


	if game.Turn == Black {
		cp = -cp
	}

	return cp
}


func EvaluateNNUEForSide(game *Game, color Color) int {
	score := EvaluateNNUE(game)
	if color == Black {
		return -score
	}
	return score
}


func GetNNUEModel() *Model {
	return nnueModel
}
