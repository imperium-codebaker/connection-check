package sites

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := map[string]Site{
		"Studio LA=remote.example.com":      {Name: "Studio LA", Host: "remote.example.com", Port: 443},
		"Studio LA=remote.example.com:4172": {Name: "Studio LA", Host: "remote.example.com", Port: 4172},
		"remote.example.com":                {Name: "remote.example.com", Host: "remote.example.com", Port: 443},
		"192.0.2.47:4172":                   {Name: "192.0.2.47", Host: "192.0.2.47", Port: 4172},
		"Studio=remote.example.com@ATL":     {Name: "Studio", Host: "remote.example.com", Port: 443, Place: "atl"},
		"remote.example.com:4172@lax":       {Name: "remote.example.com", Host: "remote.example.com", Port: 4172, Place: "lax"},
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "Name=", "host:99999", "host:abc", "host@zzz", "host@"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) accepted", bad)
		}
	}
}

func TestSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "sites.json")
	if list, err := Load(path); err != nil || list != nil {
		t.Fatalf("missing file: %v, %v", list, err)
	}
	want := []Site{{Name: "A", Host: "a.example.com", Port: 443}}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Load = %+v, %v", got, err)
	}
}
