package stretch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/imperium-codebaker/connection-check/internal/report"
)

type fixture struct {
	Note     string           `json:"note"`
	Snapshot report.Snapshot  `json:"snapshot"`
	Want     []report.Stretch `json:"want"`
}

func TestBuildRecordedRoutes(t *testing.T) {
	files, err := filepath.Glob("../../testdata/stretch/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var fx fixture
		if err := json.Unmarshal(b, &fx); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		s := fx.Snapshot
		if got := Build(s.Hops, s.Dest, s.Verdict); !reflect.DeepEqual(got, fx.Want) {
			t.Errorf("%s (%s):\n got %+v\nwant %+v", filepath.Base(f), fx.Note, got, fx.Want)
		}
	}
}

func hop(ttl int, owner string, avg float64) report.Hop {
	return report.Hop{TTL: ttl, Owner: owner, Answered: avg > 0, Avg: avg, Grade: "good"}
}

func kinds(ss []report.Stretch) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.Kind
	}
	return out
}

func TestLeadingUnknownHopsJoinTheFirstKnownStretch(t *testing.T) {
	hops := []report.Hop{hop(1, "unknown", 0), hop(2, "provider", 3)}
	got := Build(hops, hop(0, "site", 9), report.Verdict{})
	if !reflect.DeepEqual(kinds(got), []string{Provider, Site}) || got[0].FirstTTL != 1 || got[0].Routers != 2 {
		t.Errorf("got %+v", got)
	}
}

func TestUnknownRouteJoinsTheSite(t *testing.T) {
	hops := []report.Hop{hop(1, "unknown", 0), hop(2, "unknown", 0)}
	got := Build(hops, hop(0, "site", 9), report.Verdict{})
	if !reflect.DeepEqual(kinds(got), []string{Site}) || got[0].Routers != 3 {
		t.Errorf("got %+v", got)
	}
}

func TestNoLinkWhenTheSiteNeverAnswers(t *testing.T) {
	hops := []report.Hop{hop(1, "home", 1), hop(2, "provider", 3)}
	dest := report.Hop{Owner: "site"}
	if got := Build(hops, dest, report.Verdict{}); !reflect.DeepEqual(kinds(got), []string{Home, Provider, Site}) {
		t.Errorf("got %+v", got)
	}
}

func TestNoLinkWhenTheSitesOwnRouterAnswersLast(t *testing.T) {
	hops := []report.Hop{hop(1, "home", 1), hop(2, "provider", 3), hop(3, "site", 4)}
	if got := Build(hops, hop(0, "site", 60), report.Verdict{}); !reflect.DeepEqual(kinds(got), []string{Home, Provider, Site}) {
		t.Errorf("got %+v", got)
	}
}

func TestSiteBlameWithoutAHop(t *testing.T) {
	hops := []report.Hop{hop(1, "home", 1), hop(2, "provider", 3)}
	got := Build(hops, hop(0, "site", 60), report.Verdict{Owner: "site"})
	for _, s := range got {
		if s.Blamed != (s.Kind == Site) {
			t.Errorf("%s blamed = %v", s.Kind, s.Blamed)
		}
	}
}
