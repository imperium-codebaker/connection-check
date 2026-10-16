package monitor

import (
	"testing"
	"time"
)

func TestWindowSpans(t *testing.T) {
	press := time.Unix(1_700_000_000, 0)
	bad, _ := newWindow(Bad, time.Time{}, time.Time{}, press)
	if from, to := bad.span(press.Add(10 * time.Second)); !from.Equal(press.Add(-badHalf)) || !to.Equal(press.Add(10*time.Second)) {
		t.Errorf("bad window while running = %v..%v", from, to)
	}
	if _, to := bad.span(press.Add(time.Hour)); !to.Equal(press.Add(badHalf)) {
		t.Errorf("bad window should end 30 s after the press, ended %v", to)
	}
	def, _ := newWindow(Default, time.Time{}, time.Time{}, press)
	if from, _ := def.span(press); !from.Equal(press.Add(-defaultSpan)) {
		t.Errorf("default window starts %v", from)
	}
	if _, err := newWindow(Custom, press, press.Add(-time.Second), press); err == nil {
		t.Error("backwards custom window accepted")
	}
	if _, err := newWindow("nonsense", press, press, press); err == nil {
		t.Error("unknown window accepted")
	}
}
