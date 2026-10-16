package report

import (
	"encoding/json"
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
