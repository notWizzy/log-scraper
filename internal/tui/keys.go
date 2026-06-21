package tui

import tea "github.com/charmbracelet/bubbletea"

type keyAction int

const (
	keyNone keyAction = iota
	keyQuit
	keyUp
	keyDown
	keyEnter
	keyBack
	keyTab
	keyExpand
)

func parseKey(msg tea.KeyMsg) keyAction {
	switch msg.String() {
	case "q", "ctrl+c":
		return keyQuit
	case "up", "k":
		return keyUp
	case "down", "j":
		return keyDown
	case "enter", "l", "right":
		return keyEnter
	case "esc", "h", "left":
		return keyBack
	case "tab":
		return keyTab
	case " ":
		return keyExpand
	default:
		return keyNone
	}
}
