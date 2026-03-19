package main

import (
	"atc-sim/internal/config"
	"atc-sim/internal/game"
	"atc-sim/internal/logger"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	logger.Info("starting ATC Simulator")

	ebiten.SetWindowTitle("ATC Simulator")
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	cfg := config.Load()
	logger.Info("config loaded", "inputMode", cfg.InputMode, "ollamaEndpoint", cfg.Ollama.Endpoint, "model", cfg.Ollama.Model)
	g := game.NewGameWithConfig(cfg)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
