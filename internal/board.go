package internal

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Board [91]*Piece

func SetupStartingPosition(game *Game) {
	addPiece := func(pieceType PieceType, color Color, pos Position) {
		colorStr := "white"
		if color == Black {
			colorStr = "black"
		}
		typeStr := map[PieceType]string{
			Pawn:   "pawn",
			Knight: "knight",
			Bishop: "bishop",
			Rook:   "rook",
			Queen:  "queen",
			King:   "king",
		}[pieceType]

		piece := &Piece{
			Type:     pieceType,
			Color:    color,
			Image:    "assets/pieces-basic-png/" + colorStr + "-" + typeStr + ".png",
			Position: pos,
		}
		game.Board[positionToIndex[pos]] = piece

		if pieceType == King {
			if color == White {
				game.WhiteKing = pos
			} else {
				game.BlackKing = pos
			}
		}
	}



	addPiece(King, White, Position{1, -5, 4})


	addPiece(Queen, White, Position{-1, -4, 5})


	addPiece(Bishop, White, Position{0, -5, 5})
	addPiece(Bishop, White, Position{0, -4, 4})
	addPiece(Bishop, White, Position{0, -3, 3})


	addPiece(Knight, White, Position{-2, -3, 5})
	addPiece(Knight, White, Position{2, -5, 3})


	addPiece(Rook, White, Position{-3, -2, 5})
	addPiece(Rook, White, Position{3, -5, 2})


	addPiece(Pawn, White, Position{-4, -1, 5})
	addPiece(Pawn, White, Position{-3, -1, 4})
	addPiece(Pawn, White, Position{-2, -1, 3})
	addPiece(Pawn, White, Position{-1, -1, 2})
	addPiece(Pawn, White, Position{0, -1, 1})
	addPiece(Pawn, White, Position{1, -2, 1})
	addPiece(Pawn, White, Position{2, -3, 1})
	addPiece(Pawn, White, Position{3, -4, 1})
	addPiece(Pawn, White, Position{4, -5, 1})



	addPiece(King, Black, Position{1, 4, -5})


	addPiece(Queen, Black, Position{-1, 5, -4})


	addPiece(Bishop, Black, Position{0, 5, -5})
	addPiece(Bishop, Black, Position{0, 4, -4})
	addPiece(Bishop, Black, Position{0, 3, -3})


	addPiece(Knight, Black, Position{2, 3, -5})
	addPiece(Knight, Black, Position{-2, 5, -3})


	addPiece(Rook, Black, Position{3, 2, -5})
	addPiece(Rook, Black, Position{-3, 5, -2})


	addPiece(Pawn, Black, Position{4, 1, -5})
	addPiece(Pawn, Black, Position{3, 1, -4})
	addPiece(Pawn, Black, Position{2, 1, -3})
	addPiece(Pawn, Black, Position{1, 1, -2})
	addPiece(Pawn, Black, Position{0, 1, -1})
	addPiece(Pawn, Black, Position{-1, 2, -1})
	addPiece(Pawn, Black, Position{-2, 3, -1})
	addPiece(Pawn, Black, Position{-3, 4, -1})
	addPiece(Pawn, Black, Position{-4, 5, -1})
}

func DrawBoard(screen *ebiten.Image, sideLength, ox, oy float64, intensity byte) {
	curr := Position{0, 0, 0}
	curr.AddVector(Vector{DirUp, 5})
	color := 0
	palette := RoseHexPalette(intensity)


	for i := 0; i < 5; i++ {
		DrawHexagon(screen, curr, sideLength, ox, oy, palette[color%3])
		for j := 0; j < i; j++ {
			curr.AddVectors(
				Vector{DirUpRight, 1},
				Vector{DirDownRight, 1},
			)
			DrawHexagon(screen, curr, sideLength, ox, oy, palette[color%3])
		}
		curr.AddVectors(
			Vector{DirUpLeft, i},
			Vector{DirDownLeft, i + 1},
		)
		color++
	}


	for i := 0; i < 11; i++ {
		DrawHexagon(screen, curr, sideLength, ox, oy, palette[color%3])
		for j := 0; j < (5 - i%2); j++ {
			curr.AddVectors(
				Vector{DirUpRight, 1},
				Vector{DirDownRight, 1},
			)
			DrawHexagon(screen, curr, sideLength, ox, oy, palette[color%3])
		}
		if i%2 == 0 {
			curr.AddVectors(
				Vector{DirUpLeft, 5},
				Vector{DirDownLeft, 5},
				Vector{DirDownRight, 1},
			)
		} else {
			curr.AddVectors(
				Vector{DirUpLeft, 4},
				Vector{DirDownLeft, 5},
			)
		}
		color++
	}


	for i := 4; i >= 0; i-- {
		DrawHexagon(screen, curr, sideLength, ox, oy, palette[color%3])
		for j := 0; j < i; j++ {
			curr.AddVectors(
				Vector{DirUpRight, 1},
				Vector{DirDownRight, 1},
			)
			DrawHexagon(screen, curr, sideLength, ox, oy, palette[color%3])
		}
		curr.AddVectors(
			Vector{DirUpLeft, i},
			Vector{DirDownLeft, i},
			Vector{DirDownRight, 1},
		)
		color++
	}
}
