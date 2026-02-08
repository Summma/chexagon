package internal

import (
	"fmt"
	"image"
	"time"

	"github.com/ebitengine/debugui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)


type HistoryEntry struct {
	Move      Move
	BoardSnap [91]*Piece
	WhiteKing Position
	BlackKing Position
	Turn      Color
}

type Game struct {
	debugui          debugui.DebugUI
	HexagonSize      int
	ColorIntensity   int
	PieceScale       int
	WhiteValue       int
	BlackValue       int
	WhiteUseTime     bool
	BlackUseTime     bool
	WhiteUseNNUE     bool
	BlackUseNNUE     bool
	Board            Board
	ScreenWidth      int
	ScreenHeight     int
	SelectedPiece    *Piece
	WhiteKing        Position
	BlackKing        Position
	EPPosition       *Position
	EPColor          *Color
	PromotingPawn    *Piece
	PromotionPos     Position
	PromotionMove    Move
	Turn             Color
	Hash             uint64
	PawnHash         uint64
	PositionHistory  map[uint64]int
	Thinking         bool
	GameOver         bool
	PendingStats     chan SearchStats
	LastSearchStats  *SearchStats
	WhiteSearchStats *SearchStats
	BlackSearchStats *SearchStats


	MoveHistory    []HistoryEntry
	InitialBoard   [91]*Piece
	InitialWKing   Position
	InitialBKing   Position
	ViewIndex      int
	ViewingHistory bool
}

func NewGame() *Game {
	return &Game{
		HexagonSize:     38,
		ColorIntensity:  50,
		PieceScale:      50,
		WhiteValue:      0,
		BlackValue:      5,
		WhiteUseTime:    false,
		BlackUseTime:    false,
		WhiteUseNNUE:    true,
		BlackUseNNUE:    true,
		ScreenWidth:     1280,
		ScreenHeight:    960,
		Board:           [91]*Piece{},
		BlackKing:       Position{1, 4, -5},
		WhiteKing:       Position{1, -5, -4},
		Turn:            White,
		PositionHistory: make(map[uint64]int),
		PendingStats:    make(chan SearchStats, 1),
		MoveHistory:     make([]HistoryEntry, 0),
		ViewIndex:       -1,
		ViewingHistory:  false,
	}
}

func (g *Game) InitHash() {
	g.Hash = g.ComputeHash()
	g.PawnHash = g.ComputePawnHash()
	g.PositionHistory[g.Hash] = 1


	g.InitialWKing = g.WhiteKing
	g.InitialBKing = g.BlackKing
	for i, piece := range g.Board {
		if piece != nil {
			pieceCopy := *piece
			g.InitialBoard[i] = &pieceCopy
		}
	}
}


func (g *Game) RecordMove(move Move) {
	var boardSnap [91]*Piece
	for i, piece := range g.Board {
		if piece != nil {
			pieceCopy := *piece
			boardSnap[i] = &pieceCopy
		}
	}

	entry := HistoryEntry{
		Move:      move,
		BoardSnap: boardSnap,
		WhiteKing: g.WhiteKing,
		BlackKing: g.BlackKing,
		Turn:      g.Turn,
	}
	g.MoveHistory = append(g.MoveHistory, entry)


	g.ViewingHistory = false
	g.ViewIndex = -1
}


func (g *Game) GetViewBoard() [91]*Piece {
	if !g.ViewingHistory || g.ViewIndex < 0 {
		return g.Board
	}
	if g.ViewIndex == 0 {
		return g.InitialBoard
	}

	return g.MoveHistory[g.ViewIndex-1].BoardSnap
}



func (g *Game) NavigateHistory(delta int) {
	if len(g.MoveHistory) == 0 {
		return
	}



	newIndex := g.ViewIndex + delta


	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex > len(g.MoveHistory) {
		newIndex = len(g.MoveHistory)
	}

	if newIndex == len(g.MoveHistory) {

		g.ViewingHistory = false
		g.ViewIndex = -1
	} else {
		g.ViewingHistory = true
		g.ViewIndex = newIndex
	}
}

func (g *Game) IsRepetition() bool {
	return g.PositionHistory[g.Hash] >= 3
}

func (g *Game) StartSearch(depth int) {
	g.StartSearchWithConfig(SearchConfig{
		Depth:            depth,
		StopSearchByTime: false,
	})
}

func (g *Game) StartSearchWithTime(timeLimit time.Duration) {
	g.StartSearchWithConfig(SearchConfig{
		StopSearchByTime: true,
		TimeLimit:        timeLimit,
	})
}

func (g *Game) StartSearchWithConfig(config SearchConfig) {
	g.Thinking = true

	boardCopy := [91]*Piece{}
	for i, piece := range g.Board {
		if piece != nil {
			pieceCopy := *piece
			boardCopy[i] = &pieceCopy
		}
	}

	var epPosCopy *Position
	if g.EPPosition != nil {
		ep := *g.EPPosition
		epPosCopy = &ep
	}
	var epColorCopy *Color
	if g.EPColor != nil {
		ec := *g.EPColor
		epColorCopy = &ec
	}

	historyCopy := make(map[uint64]int)
	for k, v := range g.PositionHistory {
		historyCopy[k] = v
	}

	gameCopy := &Game{
		Board:           boardCopy,
		WhiteKing:       g.WhiteKing,
		BlackKing:       g.BlackKing,
		EPPosition:      epPosCopy,
		EPColor:         epColorCopy,
		Turn:            g.Turn,
		Hash:            g.Hash,
		PositionHistory: historyCopy,
	}

	go func() {
		stats := SearchWithConfig(gameCopy, config)

		if stats.Found {
			evalPawns := float64(stats.Score) / 100.0
			var evalStr string
			if stats.Score > 0 {
				evalStr = fmt.Sprintf("+%.2f", evalPawns)
			} else {
				evalStr = fmt.Sprintf("%.2f", evalPawns)
			}
			nps := float64(stats.Nodes) / stats.Duration.Seconds()
			qsPct := float64(stats.QSNodes) / float64(stats.Nodes) * 100
			fmt.Printf("Search completed: time=%v, depth=%d, qsDepth=%d, eval=%s, nodes=%d (qs=%.0f%%), nps=%.0f\n",
				stats.Duration.Round(time.Millisecond), stats.DepthReached, stats.QSDepthReached, evalStr, stats.Nodes, qsPct, nps)
		}

		g.PendingStats <- stats
	}()
}

func (g *Game) Update() error {
	inputState, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Settings", image.Rect(0, 0, 320, 400), func(layout debugui.ContainerLayout) {
			ctx.Text("Hexagon Size")
			ctx.Slider(&g.HexagonSize, 0, 45, 1)
			ctx.Text("Hexagon Color Intensity")
			ctx.Slider(&g.ColorIntensity, 0, 100, 1)
			ctx.Text("Piece Scale")
			ctx.Slider(&g.PieceScale, 0, 200, 1)

			ctx.Text("")

			var whiteLabel string
			if g.WhiteValue == 0 {
				whiteLabel = "White: Human"
			} else if g.WhiteUseTime {
				whiteLabel = fmt.Sprintf("White Time: %dms", g.WhiteValue)
			} else {
				whiteLabel = fmt.Sprintf("White Depth: %d", g.WhiteValue)
			}
			ctx.Text(whiteLabel)
			if g.WhiteUseTime {
				ctx.Slider(&g.WhiteValue, 0, 10000, 100)
			} else {
				ctx.Slider(&g.WhiteValue, 0, 10, 1)
			}
			ctx.Checkbox(&g.WhiteUseTime, "White: Use Time")
			ctx.Checkbox(&g.WhiteUseNNUE, "White: Use NNUE")

			ctx.Text("")

			var blackLabel string
			if g.BlackValue == 0 {
				blackLabel = "Black: Human"
			} else if g.BlackUseTime {
				blackLabel = fmt.Sprintf("Black Time: %dms", g.BlackValue)
			} else {
				blackLabel = fmt.Sprintf("Black Depth: %d", g.BlackValue)
			}
			ctx.Text(blackLabel)
			if g.BlackUseTime {
				ctx.Slider(&g.BlackValue, 0, 10000, 100)
			} else {
				ctx.Slider(&g.BlackValue, 0, 10, 1)
			}
			ctx.Checkbox(&g.BlackUseTime, "Black: Use Time")
			ctx.Checkbox(&g.BlackUseNNUE, "Black: Use NNUE")
		})
		return nil
	})
	if err != nil {
		return err
	}

	select {
	case stats := <-g.PendingStats:
		g.Thinking = false
		g.LastSearchStats = &stats
		if stats.SearcherColor == White {
			g.WhiteSearchStats = &stats
		} else {
			g.BlackSearchStats = &stats
		}
		if stats.Found {
			g.MakeMove(stats.BestMove)
			g.RecordMove(stats.BestMove)
			g.Turn = g.Turn.Other()
		} else {
			if !g.GameOver {
				if IsKingInCheck(g, g.Turn) {
					fmt.Println("Checkmate!")
				} else {
					fmt.Println("Stalemate!")
				}
				g.GameOver = true
			}
		}
	default:
	}

	if g.PromotingPawn == nil && !g.Thinking && !g.GameOver {
		if g.Turn == White && g.WhiteValue > 0 {
			if g.WhiteUseTime {
				g.StartSearchWithConfig(SearchConfig{
					StopSearchByTime: true,
					TimeLimit:        time.Duration(g.WhiteValue) * time.Millisecond,
					UseNNUE:          g.WhiteUseNNUE,
				})
			} else {
				g.StartSearchWithConfig(SearchConfig{
					Depth:   g.WhiteValue,
					UseNNUE: g.WhiteUseNNUE,
				})
			}
		} else if g.Turn == Black && g.BlackValue > 0 {
			if g.BlackUseTime {
				g.StartSearchWithConfig(SearchConfig{
					StopSearchByTime: true,
					TimeLimit:        time.Duration(g.BlackValue) * time.Millisecond,
					UseNNUE:          g.BlackUseNNUE,
				})
			} else {
				g.StartSearchWithConfig(SearchConfig{
					Depth:   g.BlackValue,
					UseNNUE: g.BlackUseNNUE,
				})
			}
		}
	}

	if inputState != 0 {
		return nil
	}


	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.NavigateHistory(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.NavigateHistory(1)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		if len(g.MoveHistory) > 0 {
			g.ViewingHistory = true
			g.ViewIndex = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		g.ViewingHistory = false
		g.ViewIndex = -1
	}


	if inpututil.IsKeyJustPressed(ebiten.KeyA) && !g.Thinking {
		useNNUE := true
		if g.Turn == White {
			useNNUE = g.WhiteUseNNUE
		} else {
			useNNUE = g.BlackUseNNUE
		}
		depth := 4
		AnalyzePosition(g, depth, useNNUE)
	}


	if g.ViewingHistory {
		return nil
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		if g.PromotingPawn != nil {
			selectedType := g.getPromotionSelection(x, y)
			if selectedType != nil {
				g.PromotingPawn.Type = *selectedType
				colorStr := "white"
				if g.PromotingPawn.Color == Black {
					colorStr = "black"
				}
				typeStr := map[PieceType]string{
					Knight: "knight",
					Bishop: "bishop",
					Rook:   "rook",
					Queen:  "queen",
				}[*selectedType]
				g.PromotingPawn.Image = "assets/pieces-basic-png/" + colorStr + "-" + typeStr + ".png"


				g.PromotionMove.PromotionType = selectedType
				g.RecordMove(g.PromotionMove)

				g.PromotingPawn = nil
				g.Turn = g.Turn.Other()
			}
			return nil
		}

		clickedPosition := CoordinateToPosition(
			float64(g.HexagonSize),
			float64(g.ScreenWidth)/2.0,
			float64(g.ScreenHeight)/2.0,
			float64(x),
			float64(y),
		)

		clickedPiece := GetPiece(&g.Board, clickedPosition)
		if g.SelectedPiece == nil || (clickedPiece != nil && g.SelectedPiece.Color == clickedPiece.Color) {
			if clickedPiece == nil || clickedPiece.Color != g.Turn {
				return nil
			}
			g.SelectedPiece = clickedPiece
		} else if g.SelectedPiece != nil && (clickedPiece == nil || (clickedPiece.Color != g.SelectedPiece.Color)) {
			move := GetMove(g.SelectedPiece.Position, clickedPosition, g)
			fmt.Println(move)
			if move.IsValid {
				g.MakeMove(move)

				if move.IsPromotion {
					g.PromotingPawn = GetPiece(&g.Board, clickedPosition)
					g.PromotionPos = clickedPosition
					g.PromotionMove = move
				} else {
					g.RecordMove(move)
					g.Turn = g.Turn.Other()
				}
			}
			g.SelectedPiece = nil
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	DrawBoard(screen, float64(g.HexagonSize), 640, 480, byte(g.ColorIntensity))


	if g.SelectedPiece != nil && !g.ViewingHistory {
		g.drawMoveIndicators(screen)
	}


	viewBoard := g.GetViewBoard()
	for _, piece := range viewBoard {
		if piece != nil {
			piece.Draw(screen, g)
		}
	}

	if g.PromotingPawn != nil && !g.ViewingHistory {
		g.drawPromotionUI(screen)
	}

	g.drawEvaluation(screen)


	if g.ViewingHistory || len(g.MoveHistory) > 0 {
		g.drawHistoryIndicator(screen)
	}

	g.debugui.Draw(screen)
}

func (g *Game) drawEvaluation(screen *ebiten.Image) {
	x := 10
	y := 500

	turnStr := "White to move"
	if g.Turn == Black {
		turnStr = "Black to move"
	}
	ebitenutil.DebugPrintAt(screen, turnStr, x, y)

	if g.Thinking {
		ebitenutil.DebugPrintAt(screen, "Thinking...", x, y+20)
	}

	yOffset := y + 45

	if g.WhiteValue > 0 {
		whiteEvalType := "HCE"
		if g.WhiteUseNNUE {
			whiteEvalType = "NNUE"
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("White Engine (%s):", whiteEvalType), x, yOffset)
		if g.WhiteSearchStats != nil {
			evalStr := formatEval(g.WhiteSearchStats.Score)
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("  Eval: %s, Depth: %d, QS: %d, Time: %v",
				evalStr,
				g.WhiteSearchStats.DepthReached,
				g.WhiteSearchStats.QSDepthReached,
				g.WhiteSearchStats.Duration.Round(time.Millisecond)), x, yOffset+15)
		} else {
			ebitenutil.DebugPrintAt(screen, "  No search yet", x, yOffset+15)
		}
		yOffset += 40
	}

	if g.BlackValue > 0 {
		blackEvalType := "HCE"
		if g.BlackUseNNUE {
			blackEvalType = "NNUE"
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Black Engine (%s):", blackEvalType), x, yOffset)
		if g.BlackSearchStats != nil {
			evalStr := formatEval(g.BlackSearchStats.Score)
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("  Eval: %s, Depth: %d, QS: %d, Time: %v",
				evalStr,
				g.BlackSearchStats.DepthReached,
				g.BlackSearchStats.QSDepthReached,
				g.BlackSearchStats.Duration.Round(time.Millisecond)), x, yOffset+15)
		} else {
			ebitenutil.DebugPrintAt(screen, "  No search yet", x, yOffset+15)
		}
	}
}

func formatEval(score int) string {
	evalPawns := float64(score) / 100.0
	if score > 0 {
		return fmt.Sprintf("+%.2f", evalPawns)
	}
	return fmt.Sprintf("%.2f", evalPawns)
}

func (g *Game) drawHistoryIndicator(screen *ebiten.Image) {
	x := g.ScreenWidth - 250
	y := g.ScreenHeight - 80

	totalMoves := len(g.MoveHistory)
	currentMove := totalMoves

	if g.ViewingHistory {
		currentMove = g.ViewIndex
	}


	positionText := fmt.Sprintf("Move: %d / %d", currentMove, totalMoves)
	ebitenutil.DebugPrintAt(screen, positionText, x, y)


	if g.ViewingHistory {
		ebitenutil.DebugPrintAt(screen, "[Viewing History]", x, y+15)
	}
	ebitenutil.DebugPrintAt(screen, "< > arrows to navigate", x, y+30)


	if g.ViewingHistory && g.ViewIndex > 0 {
		lastMove := g.MoveHistory[g.ViewIndex-1].Move
		moveStr := fmt.Sprintf("Last: %s %v->%v",
			lastMove.PieceType, lastMove.From, lastMove.To)
		ebitenutil.DebugPrintAt(screen, moveStr, x, y+45)
	}
}

func (g *Game) drawMoveIndicators(screen *ebiten.Image) {
	if g.SelectedPiece == nil {
		return
	}

	moves := g.SelectedPiece.GenerateMoves(g)

	smallRadius := float64(g.HexagonSize) * 0.25

	inscribedRadius := float64(g.HexagonSize) * 0.866
	ringThickness := float64(g.HexagonSize) * 0.12

	ox := float64(g.ScreenWidth) / 2.0
	oy := float64(g.ScreenHeight) / 2.0

	for _, pm := range moves {
		move := Move{
			From:          pm.From,
			To:            pm.To,
			PieceType:     g.SelectedPiece.Type,
			PieceColor:    g.SelectedPiece.Color,
			PromotionType: pm.PromotionType,
			IsValid:       true,
		}

		isCapture := false
		if GetPiece(&g.Board, pm.To) != nil {
			move.IsCapture = true
			move.ToPiece = GetPiece(&g.Board, pm.To)
			isCapture = true
		}

		if g.SelectedPiece.Type == Pawn && g.EPPosition != nil && pm.To.Equals(*g.EPPosition) {
			move.IsEnPassant = true
			move.IsCapture = true
			isCapture = true
			if g.SelectedPiece.Color == White {
				move.ToPiece = GetPiece(&g.Board, pm.To.AddReturn(DirDown))
			} else {
				move.ToPiece = GetPiece(&g.Board, pm.To.AddReturn(DirUp))
			}
		}

		undo := g.MakeMove(move)
		legal := !IsKingInCheck(g, g.SelectedPiece.Color)
		g.UnmakeMove(move, undo)

		if legal {
			if isCapture {
				DrawRing(screen, pm.To, inscribedRadius, ringThickness, ox, oy, float64(g.HexagonSize), CaptureIndicatorColor)
			} else {
				DrawCircle(screen, pm.To, smallRadius, ox, oy, float64(g.HexagonSize), MoveIndicatorColor)
			}
		}
	}
}

func (g *Game) getPromotionHexPositions() []struct {
	X, Y      float64
	PieceType PieceType
} {
	startX := float64(g.ScreenWidth) - 100.0
	startY := float64(g.ScreenHeight)/2.0 - 120.0
	spacing := float64(g.HexagonSize) * 2.2

	return []struct {
		X, Y      float64
		PieceType PieceType
	}{
		{startX, startY, Queen},
		{startX, startY + spacing, Rook},
		{startX, startY + spacing*2, Bishop},
		{startX, startY + spacing*3, Knight},
	}
}

func (g *Game) drawPromotionUI(screen *ebiten.Image) {
	positions := g.getPromotionHexPositions()
	palette := RoseHexPalette(byte(g.ColorIntensity))

	for i, pos := range positions {
		DrawHexagonAtCoord(screen, pos.X, pos.Y, float64(g.HexagonSize), palette[i%3])

		colorStr := "white"
		if g.PromotingPawn.Color == Black {
			colorStr = "black"
		}
		typeStr := map[PieceType]string{
			Knight: "knight",
			Bishop: "bishop",
			Rook:   "rook",
			Queen:  "queen",
		}[pos.PieceType]
		imagePath := "assets/pieces-basic-png/" + colorStr + "-" + typeStr + ".png"

		DrawPieceAtCoord(screen, imagePath, pos.X, pos.Y, float64(g.PieceScale)/100)
	}
}

func (g *Game) getPromotionSelection(clickX, clickY int) *PieceType {
	positions := g.getPromotionHexPositions()
	hexSize := float64(g.HexagonSize)

	for _, pos := range positions {
		dx := float64(clickX) - pos.X
		dy := float64(clickY) - pos.Y
		if dx*dx+dy*dy <= hexSize*hexSize {
			pieceType := pos.PieceType
			return &pieceType
		}
	}
	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 1280, 960
}
