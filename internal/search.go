package internal

import (
	"sync"
	"time"
)

const (
	infinity   = 1000000
	mateScore  = 100000
	maxPly     = 64
	maxQSDepth = 4
	nullMoveR  = 2
)

var TT = NewTranspositionTable(1 << 20)

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
	QSNodes        uint64
}

type searchState struct {
	startTime   time.Time
	timeLimit   time.Duration
	stopped     bool
	maxQS       int32
	useNNUE     bool
	nodes       uint64
	qsNodes     uint64
	killers     [maxPly][2]Move
	history     [2][91][91]int
	qsMoveBuf   [maxPly][64]Move
	qsScoreBuf  [maxPly][64]int
}

var searchStatePool = sync.Pool{
	New: func() interface{} {
		return &searchState{}
	},
}

func getSearchState() *searchState {
	s := searchStatePool.Get().(*searchState)
	s.stopped = false
	s.maxQS = 0
	s.nodes = 0
	s.qsNodes = 0
	for i := range s.killers {
		s.killers[i][0] = Move{}
		s.killers[i][1] = Move{}
	}
	for c := 0; c < 2; c++ {
		for i := 0; i < 91; i++ {
			for j := 0; j < 91; j++ {
				s.history[c][i][j] = 0
			}
		}
	}
	return s
}

func putSearchState(s *searchState) {
	searchStatePool.Put(s)
}

func posIndex(p Position) int {
	idx := PositionToIndex(p)
	if idx < 0 {
		return 0
	}
	return idx
}

func (s *searchState) storeKiller(ply int, move Move) {
	if ply >= maxPly {
		return
	}
	if s.killers[ply][0].From != move.From || s.killers[ply][0].To != move.To {
		s.killers[ply][1] = s.killers[ply][0]
		s.killers[ply][0] = move
	}
}

func (s *searchState) isKiller(ply int, move Move) bool {
	if ply >= maxPly {
		return false
	}
	return (s.killers[ply][0].From == move.From && s.killers[ply][0].To == move.To) ||
		(s.killers[ply][1].From == move.From && s.killers[ply][1].To == move.To)
}

func (s *searchState) updateHistory(move Move, color Color, depth int) {
	c := 0
	if color == Black {
		c = 1
	}
	from := posIndex(move.From)
	to := posIndex(move.To)
	s.history[c][from][to] += depth * depth
	if s.history[c][from][to] > 10000 {
		for i := 0; i < 91; i++ {
			for j := 0; j < 91; j++ {
				s.history[c][i][j] /= 2
			}
		}
	}
}

func (s *searchState) getHistory(move Move, color Color) int {
	c := 0
	if color == Black {
		c = 1
	}
	return s.history[c][posIndex(move.From)][posIndex(move.To)]
}

func mvvLvaScore(move *Move) int {
	if !move.IsCapture || move.ToPiece == nil {
		return 0
	}
	victimValue := pieceValues[move.ToPiece.Type]
	attackerValue := pieceValues[move.PieceType]
	return victimValue*10 - attackerValue
}

func (s *searchState) orderMoves(game *Game, moves []Move, ttMove *Move, ply int) {
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
		if s.isKiller(ply, moves[i]) {
			moves[killerEnd], moves[i] = moves[i], moves[killerEnd]
			killerEnd++
		}
	}

	if len(moves)-killerEnd > 1 {
		color := game.Turn
		for i := killerEnd + 1; i < len(moves); i++ {
			j := i
			for j > killerEnd && s.getHistory(moves[j], color) > s.getHistory(moves[j-1], color) {
				moves[j], moves[j-1] = moves[j-1], moves[j]
				j--
			}
		}
	}
}

const aspirationWindow = 50

func (s *searchState) isTimeUp() bool {
	if s.timeLimit == 0 {
		return false
	}
	return time.Since(s.startTime) >= s.timeLimit
}

func (s *searchState) evaluateForSearch(game *Game) int {
	if s.useNNUE {
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

	s := getSearchState()
	defer putSearchState(s)

	s.startTime = time.Now()
	s.useNNUE = config.UseNNUE

	if config.StopSearchByTime {
		s.timeLimit = config.TimeLimit
	} else {
		s.timeLimit = 0
	}

	searcherColor := game.Turn

	moves := GenerateAllLegalMoves(game)
	if len(moves) == 0 {
		return SearchStats{
			Duration:       time.Since(s.startTime),
			DepthReached:   0,
			QSDepthReached: 0,
			Found:          false,
			SearcherColor:  searcherColor,
			Nodes:          s.nodes,
			QSNodes:        s.qsNodes,
		}
	}

	if len(moves) == 1 {
		evalScore := Evaluate(game)
		return SearchStats{
			Duration:       time.Since(s.startTime),
			DepthReached:   1,
			QSDepthReached: 0,
			BestMove:       moves[0],
			Score:          evalScore,
			Found:          true,
			SearcherColor:  searcherColor,
			Nodes:          s.nodes,
			QSNodes:        s.qsNodes,
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
		if config.StopSearchByTime && s.isTimeUp() {
			break
		}

		var iterBestMove Move
		var iterScore int

		if d > 1 {
			alpha := score - aspirationWindow
			beta := score + aspirationWindow

			iterBestMove, iterScore = s.searchRootWithWindow(game, moves, d, alpha, beta)

			if s.stopped {
				break
			}

			if iterScore <= alpha || iterScore >= beta {
				iterBestMove, iterScore = s.searchRootWithWindow(game, moves, d, -infinity, infinity)
				if s.stopped {
					break
				}
			}
		} else {
			iterBestMove, iterScore = s.searchRootWithWindow(game, moves, d, -infinity, infinity)
			if s.stopped {
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
		Duration:       time.Since(s.startTime),
		DepthReached:   depthReached,
		QSDepthReached: int(s.maxQS),
		BestMove:       bestMove,
		Score:          displayScore,
		Found:          true,
		SearcherColor:  searcherColor,
		Nodes:          s.nodes,
		QSNodes:        s.qsNodes,
	}
}

func (s *searchState) searchRootWithWindow(game *Game, moves []Move, depth int, alpha, beta int) (Move, int) {
	bestMove := moves[0]
	best := -infinity

	for i, move := range moves {
		undo := game.MakeMove(move)
		game.Turn = game.Turn.Other()

		var score int
		if i == 0 {
			score = -s.negamax(game, -beta, -alpha, depth-1, 1, true)
		} else {
			score = -s.negamax(game, -alpha-1, -alpha, depth-1, 1, true)
			if score > alpha && score < beta {
				score = -s.negamax(game, -beta, -alpha, depth-1, 1, true)
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

func (s *searchState) negamax(game *Game, alpha, beta, depth, ply int, canNull bool) int {
	s.nodes++

	if ply <= 2 && s.timeLimit > 0 && s.isTimeUp() {
		s.stopped = true
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
		return s.quiesce(game, alpha, beta, 0)
	}

	if canNull && !inCheck && depth >= nullMoveR+1 && beta < mateScore-maxPly {
		epBefore := game.EPPosition
		hashBefore := game.Hash

		game.Turn = game.Turn.Other()
		game.EPPosition = nil
		game.Hash ^= zobristSideToMove

		score := -s.negamax(game, -beta, -beta+1, depth-1-nullMoveR, ply+1, false)

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

	s.orderMoves(game, moves, ttMove, ply)

	var futilityBase int
	canFutilityPrune := false
	if depth <= 3 && !inCheck && alpha < mateScore-maxPly && alpha > -mateScore+maxPly {
		futilityBase = s.evaluateForSearch(game)
		futilityMargin := depth * 150
		canFutilityPrune = futilityBase+futilityMargin <= alpha
	}

	var bestMove *Move
	bestScore := -infinity

	for i := range moves {
		if s.stopped {
			return alpha
		}

		move := &moves[i]

		if canFutilityPrune && i > 0 && !move.IsCapture && !move.IsPromotion && !s.isKiller(ply, *move) {
			continue
		}

		undo := game.MakeMove(*move)
		game.Turn = game.Turn.Other()

		var score int
		newDepth := depth - 1

		if inCheck && ply < maxPly-1 {
			newDepth = depth
		}

		if i >= 4 && depth >= 3 && !move.IsCapture && !s.isKiller(ply, *move) && !inCheck {
			score = -s.negamax(game, -alpha-1, -alpha, newDepth-1, ply+1, true)
			if score > alpha && !s.stopped {
				score = -s.negamax(game, -beta, -alpha, newDepth, ply+1, true)
			}
		} else if i == 0 {
			score = -s.negamax(game, -beta, -alpha, newDepth, ply+1, true)
		} else {
			score = -s.negamax(game, -alpha-1, -alpha, newDepth, ply+1, true)
			if score > alpha && score < beta && !s.stopped {
				score = -s.negamax(game, -beta, -alpha, newDepth, ply+1, true)
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
				s.storeKiller(ply, *move)
				s.updateHistory(*move, game.Turn, depth)
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

func (s *searchState) quiesce(game *Game, alpha, beta, qsDepth int) int {
	s.nodes++
	s.qsNodes++

	if int32(qsDepth) > s.maxQS {
		s.maxQS = int32(qsDepth)
	}

	standPat := s.evaluateForSearch(game)

	if standPat >= beta {
		return beta
	}
	if standPat > alpha {
		alpha = standPat
	}

	if qsDepth >= maxQSDepth {
		return alpha
	}

	if standPat+1000 < alpha {
		return alpha
	}

	turn := game.Turn
	numMoves := 0
	moves := &s.qsMoveBuf[qsDepth]
	scores := &s.qsScoreBuf[qsDepth]

	for _, piece := range game.Board {
		if piece == nil || piece.Color != turn {
			continue
		}

		pieceType := piece.Type
		attackerVal := pieceValues[pieceType]

		plainMoves := piece.GenerateMoves(game)
		for _, pm := range plainMoves {
			target := GetPiece(&game.Board, pm.To)
			isEP := false

			if target == nil {
				if pieceType == Pawn && game.EPPosition != nil && pm.To.Equals(*game.EPPosition) {
					isEP = true
					if turn == White {
						target = GetPiece(&game.Board, pm.To.AddReturn(DirDown))
					} else {
						target = GetPiece(&game.Board, pm.To.AddReturn(DirUp))
					}
				} else {
					continue
				}
			} else if target.Color == turn {
				continue
			}

			targetVal := pieceValues[target.Type]

			if standPat+targetVal+300 < alpha {
				continue
			}

			if attackerVal > targetVal+100 && pieceType != Queen && pieceType != Pawn {
				continue
			}

			if numMoves < 64 {
				moves[numMoves] = Move{
					From:        pm.From,
					To:          pm.To,
					PieceType:   pieceType,
					PieceColor:  turn,
					IsCapture:   true,
					ToPiece:     target,
					IsEnPassant: isEP,
					IsValid:     true,
				}
				scores[numMoves] = targetVal*10 - attackerVal
				numMoves++
			}
		}
	}

	for i := 0; i < numMoves; i++ {
		bestIdx := i
		bestScore := scores[i]
		for j := i + 1; j < numMoves; j++ {
			if scores[j] > bestScore {
				bestIdx = j
				bestScore = scores[j]
			}
		}
		if bestIdx != i {
			moves[i], moves[bestIdx] = moves[bestIdx], moves[i]
			scores[i], scores[bestIdx] = scores[bestIdx], scores[i]
		}

		move := &moves[i]

		undo := game.MakeMove(*move)
		if IsKingInCheck(game, turn) {
			game.UnmakeMove(*move, undo)
			continue
		}
		game.Turn = turn.Other()

		score := -s.quiesce(game, -beta, -alpha, qsDepth+1)

		game.Turn = turn
		game.UnmakeMove(*move, undo)

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	return alpha
}
