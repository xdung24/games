package bombsweeper

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	timeCounterSize = 3
)

// TimeCounter counts the time
type TimeCounter struct {
	*fyne.Container

	digits  []*widget.Label
	current int
}

// NewTimeCounter creates a new time counter
func NewTimeCounter(init int) *TimeCounter {
	numd := getDigits(init)
	d := make([]*widget.Label, timeCounterSize)
	c := container.NewHBox()
	for i := 0; i < timeCounterSize; i++ {
		h := widget.NewLabel(strconv.FormatUint(uint64(numd[i]), 10))
		h.TextStyle.Bold = true

		d[i] = h
		c.Add(h)
	}

	tc := &TimeCounter{
		Container: c,
		digits:    d,
		current:   init,
	}
	return tc
}

// Increment updates the counter
func (tc *TimeCounter) Increment() {
	tc.current++
	numd := getDigits(tc.current)
	for i := 0; i < timeCounterSize; i++ {
		tc.digits[i].SetText(strconv.FormatUint(uint64(numd[i]), 10))
	}
}
