package internal

import (
	"sync/atomic"
	"time"
)

const (
	infinity   = 1000000
	mateScore  = 100000
	maxPly     = 64
	maxQSDepth = 8
	nullMoveR  = 2
)

var TT = NewTranspositionTable(1 << 20)

var killers [maxPly][2]Move
var history [2][91][91]int

type SearchConfig struct {
	Depth            int
	StopSearchByTime bool
	TimeLimit        time.Duration
	UseNNUE          bool
}

type SearchStats struct {
	Duration       time.Duration
	DepthReached   int
	QSDepthReached int
	BestMove       Move
	Score          int
	Found          bool
	SearcherColor  Color
	Nodes          uint64
}

var (
	searchStartTime time.Time
	searchTimeLimit time.Duration
	searchStopped   atomic.Bool
	maxQSReached    atomic.Int32
	searchUseNNUE   bool
	nodesSearched   atomic.Uint64
)

func posIndex(p Position) int {
	idx := PositionToIndex(p)
	if idx < 0 {
		return 0
	}
	return idx
}

func clearKillers() {
	for i := range killers {
		killers[i][0] = Move{}
		killers[i][1] = Move{}
	}
}

func clearHistory() {
	for c := 0; c < 2; c++ {
		for i := 0; i < 91; i++ {
			for j := 0; j < 91; j++ {
				history[c][i][j] = 0
			}
		}
	}
}

func storeKiller(ply int, move Move) {
	if ply >= maxPly {
		return
	}
	if killers[ply][0].From != move.From || killers[ply][0].To != move.To {
		killers[ply][1] = killers[ply][0]
		killers[ply][0] = move
	}
}

func isKiller(ply int, move Move) bool {
	if ply >= maxPly {
		return false
	}
	return (killers[ply][0].From == move.From && killers[ply][0].To == move.To) ||
		(killers[ply][1].From == move.From && killers[ply][1].To == move.To)
}

func updateHistory(move Move, color Color, depth int) {
	c := 0
	if color == Black {
		c = 1
	}
	from := posIndex(move.From)
	to := posIndex(move.To)
	history[c][from][to] += depth * depth
	if history[c][from][to] > 10000 {
		for i := 0; i < 91; i++ {
			for j := 0; j < 91; j++ {
				history[c][i][j] /= 2
			}
		}
	}
}

func getHistory(move Move, color Color) int {
	c := 0
	if color == Black {
		c = 1
	}
	return history[c][posIndex(move.From)][posIndex(move.To)]
}

func mvvLvaScore(move *Move) int {
	if !move.IsCapture || move.ToPiece == nil {
		return 0
	}
	victimValue := pieceValues[move.ToPiece.Type]
	attackerValue := pieceValues[move.PieceType]
	return victimValue*10 - attackerValue
}

func orderMoves(game *Game, moves []Move, ttMove *Move, ply int) {
	if ttMove != nil {
		for i := range moves {
			if moves[i].From == ttMove.From && moves[i].To == ttMove.To {
				moves[0], moves[i] = moves[i], moves[0]
				break
			}
		}
	}

	start := 0
	if ttMove != nil {
		start = 1
	}

	captureEnd := start
	for i := start; i < len(moves); i++ {
		if moves[i].IsCapture {
			moves[captureEnd], moves[i] = moves[i], moves[captureEnd]
			captureEnd++
		}
	}

	for i := start + 1; i < captureEnd; i++ {
		j := i
		for j > start && mvvLvaScore(&moves[j]) > mvvLvaScore(&moves[j-1]) {
			moves[j], moves[j-1] = moves[j-1], moves[j]
			j--
		}
	}

	killerEnd := captureEnd
	for i := captureEnd; i < len(moves); i++ {
		if isKiller(ply, moves[i]) {
			moves[killerEnd], moves[i] = moves[i], moves[killerEnd]
			killerEnd++
		}
	}

	if len(moves)-killerEnd > 1 {
		color := game.Turn
		for i := killerEnd + 1; i < len(moves); i++ {
			j := i
			for j > killerEnd && getHistory(moves[j], color) > getHistory(moves[j-1], color) {
				moves[j], moves[j-1] = moves[j-1], moves[j]
				j--
			}
		}
	}
}

const aspirationWindow = 50

func isTimeUp() bool {
	if searchTimeLimit == 0 {
		return false
	}
	return time.Since(searchStartTime) >= searchTimeLimit
}

// evaluateForSearch returns evaluation from side-to-move perspective
// Uses NNUE or handcrafted based on searchUseNNUE setting
func evaluateForSearch(game *Game) int {
	if searchUseNNUE {
		return EvaluateNNUEForSide(game, game.Turn)
	}
	return EvaluateForSide(game, game.Turn)
}

func Search(game *Game, depth int) (Move, bool) {
	stats := SearchWithConfig(game, SearchConfig{
		Depth:            depth,
		StopSearchByTime: false,
	})
	return stats.BestMove, stats.Found
}

func SearchWithConfig(game *Game, config SearchConfig) SearchStats {
	TT.Clear()
	clearKillers()
	clearHistory()

	searchStartTime = time.Now()
	searchStopped.Store(false)
	maxQSReached.Store(0)
	nodesSearched.Store(0)
	searchUseNNUE = config.UseNNUE

	if config.StopSearchByTime {
		searchTimeLimit = config.TimeLimit
	} else {
		searchTimeLimit = 0
	}

	searcherColor := game.Turn

	moves := GenerateAllLegalMoves(game)
	if len(moves) == 0 {
		return SearchStats{
			Duration:       time.Since(searchStartTime),
			DepthReached:   0,
			QSDepthReached: 0,
			Found:          false,
			SearcherColor:  searcherColor,
			Nodes:          nodesSearched.Load(),
		}
	}

	if len(moves) == 1 {
		evalScore := Evaluate(game)
		return SearchStats{
			Duration:       time.Since(searchStartTime),
			DepthReached:   1,
			QSDepthReached: 0,
			BestMove:       moves[0],
			Score:          evalScore,
			Found:          true,
			SearcherColor:  searcherColor,
			Nodes:          nodesSearched.Load(),
		}
	}

	var bestMove Move
	score := 0
	depthReached := 0

	maxDepth := config.Depth
	if config.StopSearchByTime {
		maxDepth = 100
	}

	for d := 1; d <= maxDepth; d++ {
		if config.StopSearchByTime && isTimeUp() {
			break
		}

		var iterBestMove Move
		var iterScore int

		if d > 1 {
			alpha := score - aspirationWindow
			beta := score + aspirationWindow

			iterBestMove, iterScore = searchRootWithWindow(game, moves, d, alpha, beta)

			if searchStopped.Load() {
				break
			}

			if iterScore <= alpha || iterScore >= beta {
				iterBestMove, iterScore = searchRootWithWindow(game, moves, d, -infinity, infinity)
				if searchStopped.Load() {
					break
				}
			}
		} else {
			iterBestMove, iterScore = searchRootWithWindow(game, moves, d, -infinity, infinity)
			if searchStopped.Load() {
				break
			}
		}

		bestMove = iterBestMove
		score = iterScore
		depthReached = d

		for i := range moves {
			if moves[i].From == bestMove.From && moves[i].To == bestMove.To {
				moves[0], moves[i] = moves[i], moves[0]
				break
			}
		}
	}

	displayScore := score
	if game.Turn == Black {
		displayScore = -score
	}

	return SearchStats{
		Duration:       time.Since(searchStartTime),
		DepthReached:   depthReached,
		QSDepthReached: int(maxQSReached.Load()),
		BestMove:       bestMove,
		Score:          displayScore,
		Found:          true,
		SearcherColor:  searcherColor,
		Nodes:          nodesSearched.Load(),
	}
}

func searchRootWithWindow(game *Game, moves []Move, depth int, alpha, beta int) (Move, int) {
	bestMove := moves[0]
	best := -infinity

	for i, move := range moves {
		undo := game.MakeMove(move)
		game.Turn = game.Turn.Other()

		var score int
		if i == 0 {
			score = -negamax(game, -beta, -alpha, depth-1, 1, true)
		} else {
			score = -negamax(game, -alpha-1, -alpha, depth-1, 1, true)
			if score > alpha && score < beta {
				score = -negamax(game, -beta, -alpha, depth-1, 1, true)
			}
		}

		game.Turn = game.Turn.Other()
		game.UnmakeMove(move, undo)

		if score > best {
			best = score
			bestMove = move
			if score > alpha {
				alpha = score
			}
		}
	}

	TT.Store(game.Hash, depth, best, TTExact, &bestMove)
	return bestMove, best
}

func negamax(game *Game, alpha, beta, depth, ply int, canNull bool) int {
	nodesSearched.Add(1)

	if ply <= 2 && searchTimeLimit > 0 && isTimeUp() {
		searchStopped.Store(true)
		return 0
	}

	alphaOrig := alpha

	if game.PositionHistory[game.Hash] >= 2 {
		return 0
	}

	entry, found := TT.Probe(game.Hash)
	if found && entry.Depth >= depth {
		switch entry.Flag {
		case TTExact:
			return entry.Score
		case TTLowerBound:
			if entry.Score > alpha {
				alpha = entry.Score
			}
		case TTUpperBound:
			if entry.Score < beta {
				beta = entry.Score
			}
		}

		if alpha >= beta {
			return entry.Score
		}
	}

	inCheck := IsKingInCheck(game, game.Turn)

	if depth <= 0 {
		return quiesce(game, alpha, beta, 0)
	}

	if canNull && !inCheck && depth >= nullMoveR+1 && beta < mateScore-maxPly {
		epBefore := game.EPPosition
		hashBefore := game.Hash

		game.Turn = game.Turn.Other()
		game.EPPosition = nil
		game.Hash ^= zobristSideToMove

		score := -negamax(game, -beta, -beta+1, depth-1-nullMoveR, ply+1, false)

		game.Hash = hashBefore
		game.EPPosition = epBefore
		game.Turn = game.Turn.Other()

		if score >= beta {
			return beta
		}
	}

	var ttMove *Move
	if found && entry.BestMove != nil {
		ttMove = entry.BestMove
	}

	moves := GenerateAllLegalMoves(game)

	if len(moves) == 0 {
		if inCheck {
			return -mateScore + ply
		}
		return 0
	}

	orderMoves(game, moves, ttMove, ply)

	var futilityBase int
	canFutilityPrune := false
	if depth <= 3 && !inCheck && alpha < mateScore-maxPly && alpha > -mateScore+maxPly {
		futilityBase = evaluateForSearch(game)
		futilityMargin := depth * 150
		canFutilityPrune = futilityBase+futilityMargin <= alpha
	}

	var bestMove *Move
	bestScore := -infinity

	for i := range moves {
		if searchStopped.Load() {
			return alpha
		}

		move := &moves[i]

		if canFutilityPrune && i > 0 && !move.IsCapture && !move.IsPromotion && !isKiller(ply, *move) {
			continue
		}

		undo := game.MakeMove(*move)
		game.Turn = game.Turn.Other()

		var score int
		newDepth := depth - 1

		if inCheck && ply < maxPly-1 {
			newDepth = depth
		}

		if i >= 4 && depth >= 3 && !move.IsCapture && !isKiller(ply, *move) && !inCheck {
			score = -negamax(game, -alpha-1, -alpha, newDepth-1, ply+1, true)
			if score > alpha && !searchStopped.Load() {
				score = -negamax(game, -beta, -alpha, newDepth, ply+1, true)
			}
		} else if i == 0 {
			score = -negamax(game, -beta, -alpha, newDepth, ply+1, true)
		} else {
			score = -negamax(game, -alpha-1, -alpha, newDepth, ply+1, true)
			if score > alpha && score < beta && !searchStopped.Load() {
				score = -negamax(game, -beta, -alpha, newDepth, ply+1, true)
			}
		}

		game.Turn = game.Turn.Other()
		game.UnmakeMove(*move, undo)

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		if score >= beta {
			if !move.IsCapture {
				storeKiller(ply, *move)
				updateHistory(*move, game.Turn, depth)
			}
			TT.Store(game.Hash, depth, beta, TTLowerBound, bestMove)
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	var flag TTFlag
	if bestScore <= alphaOrig {
		flag = TTUpperBound
	} else {
		flag = TTExact
	}

	TT.Store(game.Hash, depth, bestScore, flag, bestMove)

	return alpha
}

func quiesce(game *Game, alpha, beta, qsDepth int) int {
	nodesSearched.Add(1)

	if current := maxQSReached.Load(); int32(qsDepth) > current {
		maxQSReached.Store(int32(qsDepth))
	}

	standPat := evaluateForSearch(game)

	if standPat >= beta {
		return beta
	}
	if standPat > alpha {
		alpha = standPat
	}

	if qsDepth >= maxQSDepth {
		return alpha
	}

	for _, piece := range game.Board {
		if piece == nil || piece.Color != game.Turn {
			continue
		}

		plainMoves := piece.GenerateMoves(game)
		for _, pm := range plainMoves {
			target := GetPiece(&game.Board, pm.To)
			if target == nil {
				if piece.Type == Pawn && game.EPPosition != nil && pm.To.Equals(*game.EPPosition) {
				} else {
					continue
				}
			} else if target.Color == piece.Color {
				continue
			}

			if target != nil && PieceValue(target)+200 < alpha-standPat {
				continue
			}

			move := Move{
				From:       pm.From,
				To:         pm.To,
				PieceType:  piece.Type,
				PieceColor: piece.Color,
				IsCapture:  true,
				ToPiece:    target,
				IsValid:    true,
			}

			if piece.Type == Pawn && game.EPPosition != nil && pm.To.Equals(*game.EPPosition) {
				move.IsEnPassant = true
				if piece.Color == White {
					move.ToPiece = GetPiece(&game.Board, pm.To.AddReturn(DirDown))
				} else {
					move.ToPiece = GetPiece(&game.Board, pm.To.AddReturn(DirUp))
				}
			}

			undo := game.MakeMove(move)
			if IsKingInCheck(game, piece.Color) {
				game.UnmakeMove(move, undo)
				continue
			}
			game.Turn = game.Turn.Other()

			score := -quiesce(game, -beta, -alpha, qsDepth+1)

			game.Turn = game.Turn.Other()
			game.UnmakeMove(move, undo)

			if score >= beta {
				return beta
			}
			if score > alpha {
				alpha = score
			}
		}
	}

	return alpha
}
