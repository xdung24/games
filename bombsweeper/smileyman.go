package bombsweeper

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// The different states of a game
const (
	// Are ya winnin' son?
	GameStateWin = iota
	GameStateLose
	GameStateOngoing
)

// GameState represents the current state of the game
type GameState int

// SmileyMan controls everything. God, you mean? No.
type SmileyMan struct {
	widget.Icon

	gameState GameState

	resetHandler func()
}

// NewSmileyMan creates a new smileyman
func NewSmileyMan(state GameState, resetHandler func()) *SmileyMan {
	sm := &SmileyMan{gameState: state, resetHandler: resetHandler}
	sm.ExtendBaseWidget(sm)
	sm.SetResource(getResourceByGameState(state))

	return sm
}

func (sm *SmileyMan) MinSize() fyne.Size {
	return fyne.NewSize(tileSize, tileSize)
}

// SetState chagnes smiley amns state based on game state
func (sm *SmileyMan) SetState(state GameState) {
	sm.SetResource(getResourceByGameState(state))
}

func (sm *SmileyMan) Tapped(*fyne.PointEvent) {
	sm.SetResource(getResourceByGameState(GameStateOngoing))
	sm.resetHandler()
}

func getResourceByGameState(state GameState) fyne.Resource {
	var res fyne.Resource

	switch state {
	case GameStateLose:
		res = resourceFaceLosePng
	case GameStateWin:
		res = resourceFaceWinPng
	case GameStateOngoing:
		res = resourceFaceUnpressedPng
	default:
		res = resourceFaceUnpressedPng
	}

	return res
}
