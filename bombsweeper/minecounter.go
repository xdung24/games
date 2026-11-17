package bombsweeper

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	mineCounterSize = 3
)

// MineCounter counts the possible remaining mines
type MineCounter struct {
	*fyne.Container

	digits  []*widget.Label
	current int
}

// NewMineCounter reates a new mine counter
func NewMineCounter(init int) *MineCounter {
	numd := getDigits(init)
	d := make([]*widget.Label, mineCounterSize)
	c := container.NewHBox()
	for i := 0; i < mineCounterSize; i++ {
		h := widget.NewLabel(strconv.FormatUint(uint64(numd[i]), 10))
		h.TextStyle.Bold = true

		d[i] = h
		c.Add(h)
	}

	mc := &MineCounter{
		Container: c,
		digits:    d,
		current:   init,
	}
	return mc
}

// Decrement updates the counter
func (mc *MineCounter) Decrement() {
	mc.current--
	if mc.current < 0 {
		mc.current = 0
	}
	numd := getDigits(mc.current)
	for i := 0; i < mineCounterSize; i++ {
		mc.digits[i].SetText(strconv.FormatUint(uint64(numd[i]), 10))
	}
}

// Increment updates the counter
func (mc *MineCounter) Increment() {
	mc.current++
	if mc.current < 0 {
		mc.current = 0
	}
	numd := getDigits(mc.current)
	for i := 0; i < mineCounterSize; i++ {
		mc.digits[i].SetText(strconv.FormatUint(uint64(numd[i]), 10))
	}
}

func getDigits(number int) []uint {
	digits := make([]uint, mineCounterSize)
	for i := 0; i < mineCounterSize; i++ {
		digits[mineCounterSize-i-1] = uint(number % 10)
		number /= 10
	}

	return digits
}
