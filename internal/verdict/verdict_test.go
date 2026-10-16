package verdict

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/johncbrubaker/connection-check/internal/owner"
)

type traceCase struct {
	Note  string `json:"note"`
	Input Input  `json:"input"`
	Want  struct {
		Grade    Grade       `json:"grade"`
		Measure  Measure     `json:"measure"`
		Owner    owner.Owner `json:"owner"`
		BlameTTL int         `json:"blameTtl"`
		LongRoad bool        `json:"longRoad"`
	} `json:"want"`
}

func TestJudgeRecordedTraces(t *testing.T) {
	files, _ := filepath.Glob("../../testdata/verdict/*.json")
	if len(files) == 0 {
		t.Fatal("no recorded traces found")
	}
	for _, f := range files {
		t.Run(strings.TrimSuffix(filepath.Base(f), ".json"), func(t *testing.T) {
			c := loadCase(t, f)
			v := Judge(c.Input)
			w := c.Want
			if v.Grade != w.Grade || v.Measure != w.Measure || v.Owner != w.Owner ||
				v.BlameTTL != w.BlameTTL || v.LongRoad != w.LongRoad {
				t.Errorf("%s\n got  %s %s owner=%q hop=%d longRoad=%v\n want %s %s owner=%q hop=%d longRoad=%v",
					c.Note, v.Grade, v.Measure, v.Owner, v.BlameTTL, v.LongRoad,
					w.Grade, w.Measure, w.Owner, w.BlameTTL, w.LongRoad)
			}
			if v.Headline == "" || v.Action == "" {
				t.Error("verdict has no wording")
			}
		})
	}
}

func loadCase(t *testing.T, path string) traceCase {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var c traceCase
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestGradeOfUsesHPBands(t *testing.T) {
	cases := []struct {
		m    Measure
		v    float64
		want Grade
	}{
		{Latency, 50, Good}, {Latency, 50.1, Fair}, {Latency, 100.1, Poor},
		{Loss, 0.25, Good}, {Loss, 0.33, Fair}, {Loss, 0.51, Poor},
		{Jitter, 10, Good}, {Jitter, 30.1, Poor},
	}
	for _, c := range cases {
		if got := GradeOf(c.m, c.v); got != c.want {
			t.Errorf("GradeOf(%s, %v) = %s, want %s", c.m, c.v, got, c.want)
		}
	}
}

func TestLongRoadWordingNamesTheSite(t *testing.T) {
	c := loadCase(t, "../../testdata/verdict/long-road.json")
	v := Judge(c.Input)
	if !strings.Contains(v.Action, "Studio LA") || !strings.Contains(v.Headline, "62 ms") {
		t.Errorf("wording: %q / %q", v.Headline, v.Action)
	}
}

func TestHopGradesIgnoreTroubleThatDoesNotCarry(t *testing.T) {
	hops := []Hop{
		{TTL: 1, Answered: true, Sent: 100, Loss: 64, Avg: 0.2},
		{TTL: 2, Answered: false},
		{TTL: 3, Answered: true, Sent: 100, Loss: 0, Avg: 120, Jitter: 40},
		{TTL: 4, Answered: true, Sent: 100, Loss: 0, Avg: 6},
	}
	dest := Hop{Answered: true, Sent: 500, Loss: 0, Avg: 54}
	got := HopGrades(hops, dest)
	want := []Grade{Good, "", Good, Good}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("hop %d graded %q, want %q", i+1, got[i], want[i])
		}
	}
	dest.Loss = 2
	hops[3].Loss = 2
	if g := HopGrades(hops, dest)[3]; g != Poor {
		t.Errorf("loss that reaches the destination graded %q at its first hop", g)
	}
}
