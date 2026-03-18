package main

import (
	"atc-sim/internal/config"
	"atc-sim/internal/game"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowTitle("ATC Simulator")
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	cfg := config.Load()
	g := game.NewGameWithConfig(cfg)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
