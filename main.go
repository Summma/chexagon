package main

import (
	"log"

	"chexagon/internal"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	internal.SetAssets(Assets)

	game := internal.NewGame()
	internal.SetupStartingPosition(game)
	game.InitHash()

	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowTitle("Hexagonal Chess")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
