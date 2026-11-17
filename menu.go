package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const gamesPerPage = 10

type gameMenuButton struct {
	widget.Button
	navigate func(fyne.KeyName) bool
}

func (b *gameMenuButton) TypedKey(e *fyne.KeyEvent) {
	if !b.navigate(e.Name) {
		b.Button.TypedKey(e)
	}
}

func (m *gameMenu) newPageButton(text string, delta int) *gameMenuButton {
	b := &gameMenuButton{
		Button: widget.Button{
			Text:     text,
			OnTapped: func() { m.changePage(delta) },
		},
		navigate: m.navigate,
	}
	b.ExtendBaseWidget(b)
	return b
}

type gameMenu struct {
	content            *fyne.Container
	entries            []gameEntry
	page               int
	list               *fyne.Container
	previous           *gameMenuButton
	next               *gameMenuButton
	status             *widget.Label
	selected           int
	scroll             *container.Scroll
	canvas             fyne.Canvas
	previousKeyHandler func(*fyne.KeyEvent)
}

func newGameMenu(entries []gameEntry) *gameMenu {
	m := &gameMenu{
		entries: entries,
		list:    container.NewVBox(),
		status:  widget.NewLabel(""),
	}
	m.previous = m.newPageButton("Previous", -1)
	m.next = m.newPageButton("Next", 1)
	m.scroll = container.NewVScroll(m.list)

	icon := canvas.NewImageFromResource(homeIcon)
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(96, 96))
	subtitle := widget.NewLabelWithStyle("Choose a game to play", fyne.TextAlignCenter, fyne.TextStyle{})
	pages := container.NewCenter(container.NewHBox(m.previous, m.status, m.next))
	m.content = container.NewPadded(container.NewBorder(
		container.NewVBox(container.NewCenter(icon), subtitle), pages, nil, nil, m.scroll,
	))
	m.refresh()
	return m
}

func (m *gameMenu) refresh() {
	pages := max(1, (len(m.entries)+gamesPerPage-1)/gamesPerPage)
	start := m.page * gamesPerPage
	end := min(start+gamesPerPage, len(m.entries))
	m.list.RemoveAll()
	for i, entry := range m.entries[start:end] {
		card := newGameCard(entry)
		card.onNavigate = m.navigate
		card.onFocus = func() {
			m.selected = i
			m.scrollToSelected()
		}
		card.play = func() {
			m.selected = i
			m.Hide()
			entry.play()
		}
		m.list.Add(card)
	}
	m.selected = max(0, min(m.selected, end-start-1))
	if len(m.entries) == 0 {
		m.list.Add(widget.NewLabel("No games available."))
	}
	m.status.SetText(fmt.Sprintf("Page %d of %d", m.page+1, pages))
	m.previous.Disable()
	m.next.Disable()
	if m.page > 0 {
		m.previous.Enable()
	}
	if m.page+1 < pages {
		m.next.Enable()
	}
	m.scroll.Refresh()
	m.focusSelected()
}

func (m *gameMenu) Show(w fyne.Window) {
	m.Hide()
	m.canvas = w.Canvas()
	w.SetContent(m.content)
	m.previousKeyHandler = m.canvas.OnTypedKey()
	m.canvas.SetOnTypedKey(func(e *fyne.KeyEvent) {
		if !m.navigate(e.Name) && m.previousKeyHandler != nil {
			m.previousKeyHandler(e)
		}
	})
	m.focusSelected()
}

func (m *gameMenu) Hide() {
	if m.canvas != nil {
		m.canvas.SetOnTypedKey(m.previousKeyHandler)
		m.canvas = nil
		m.previousKeyHandler = nil
	}
}

func (m *gameMenu) navigate(key fyne.KeyName) bool {
	if len(m.entries) == 0 {
		return false
	}
	switch key {
	case fyne.KeyLeft:
		m.changePage(-1)
	case fyne.KeyRight:
		m.changePage(1)
	case fyne.KeyUp:
		m.selected = max(0, m.selected-1)
		m.focusSelected()
	case fyne.KeyDown:
		m.selected = min(len(m.list.Objects)-1, m.selected+1)
		m.focusSelected()
	default:
		return false
	}
	return true
}

func (m *gameMenu) changePage(delta int) {
	pages := max(1, (len(m.entries)+gamesPerPage-1)/gamesPerPage)
	page := max(0, min(m.page+delta, pages-1))
	if page != m.page {
		m.page = page
		m.refresh()
	}
}

func (m *gameMenu) focusSelected() {
	if m.canvas == nil || len(m.entries) == 0 {
		return
	}
	m.canvas.Focus(m.list.Objects[m.selected].(*gameCard))
	m.scrollToSelected()
}

func (m *gameMenu) scrollToSelected() {
	if len(m.entries) == 0 {
		return
	}
	card := m.list.Objects[m.selected]
	top := card.Position().Y
	bottom := top + card.Size().Height
	offset := m.scroll.Offset
	if top < offset.Y {
		offset.Y = top
	} else if bottom > offset.Y+m.scroll.Size().Height {
		offset.Y = bottom - m.scroll.Size().Height
	}
	m.scroll.ScrollToOffset(offset)
}
