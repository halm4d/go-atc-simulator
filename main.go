package main

import (
	"atc-sim/internal/game"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// Set window title
	ebiten.SetWindowTitle("ATC Simulator - KJFK")
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Create and run game
	g := game.NewGame()

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
