package report

import (
	"encoding/json"
	"math/rand/v2"
	"os"
	"reflect"
	"strings"
	"testing"
)

func exampleSnapshot(t *testing.T) Snapshot {
	t.Helper()
	b, err := os.ReadFile("../../testdata/stream_example.json")
	if err != nil {
		t.Fatal(err)
	}
	var stream struct{ Sites []Snapshot }
	if err := json.Unmarshal(b, &stream); err != nil {
		t.Fatal(err)
	}
	return stream.Sites[0]
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	s := exampleSnapshot(t)
	enc, err := Encode(s)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(enc)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, s) {
		t.Fatal("decoded snapshot differs from the original")
	}
	if len(enc) > 6000 {
		t.Errorf("encoded report is %d characters; links this long break in some chat apps", len(enc))
	}
}

func TestLinkCarriesReportInFragment(t *testing.T) {
	link, err := Link("https://example.github.io/connection-check/", exampleSnapshot(t))
	if err != nil || !strings.HasPrefix(link, "https://example.github.io/connection-check/#r=") {
		t.Fatalf("link = %q, err = %v", link, err)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	if _, err := Decode("not-a-report"); err == nil {
		t.Fatal("garbage decoded without error")
	}
}

func TestTextListsEveryHop(t *testing.T) {
	s := exampleSnapshot(t)
	text := Text(s)
	for _, want := range []string{"Studio LA", "FAIR", "doesn't answer pings", "end", s.Hops[1].Addr} {
		if !strings.Contains(text, want) {
			t.Errorf("summary lacks %q:\n%s", want, text)
		}
	}
}

// fixtureSnapshot loads a stretch fixture with its stretches filled in.
func fixtureSnapshot(t *testing.T, name string) Snapshot {
	t.Helper()
	b, err := os.ReadFile("../../testdata/stretch/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Snapshot Snapshot
		Want     []Stretch
	}
	if err := json.Unmarshal(b, &fx); err != nil {
		t.Fatal(err)
	}
	fx.Snapshot.Stretches = fx.Want
	return fx.Snapshot
}

func TestTextPrintsStretchesAndRouteChanges(t *testing.T) {
	s := fixtureSnapshot(t, "private-link")
	s.RouteChanges = []RouteChange{{T: s.To - 60_000, TTL: 7, From: "203.0.113.9", To: "203.0.113.10"}}
	text := Text(s)
	for _, want := range []string{"Route:", "Private link", "Los Angeles to Atlanta", "+48 ms, no routers show",
		"Handoff: Sample, Backbone", "(no routers answer)", "ignores pings", "hop 7: 203.0.113.9 -> 203.0.113.10"} {
		if !strings.Contains(text, want) {
			t.Errorf("summary lacks %q:\n%s", want, text)
		}
	}
}

// A full hour of history, the most route changes and a noisy route must still fit in a link.
func TestLinkWithAFullHourStaysShort(t *testing.T) {
	s := fixtureSnapshot(t, "private-link")
	rnd := rand.New(rand.NewPCG(1, 2))
	for i := range 360 {
		s.Timeline = append(s.Timeline, Bucket{T: s.To - int64(360-i)*10_000, Sent: 50, Lost: rnd.IntN(3),
			Avg: float64(int(500+rnd.IntN(400))) / 10, Jitter: float64(rnd.IntN(90)) / 10})
	}
	for i := range 20 {
		s.RouteChanges = append(s.RouteChanges, RouteChange{T: s.To - int64(i)*90_000, TTL: 6 + i%3,
			From: "203.0.113.1", To: "203.0.113.2"})
	}
	enc, err := Encode(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) > 8000 {
		t.Errorf("encoded report is %d characters; links this long break in some chat apps", len(enc))
	}
	t.Logf("a full-hour report link carries %d characters", len(enc))
}
