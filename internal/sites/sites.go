// Package sites parses, saves and loads the list of remote hosts to check.
package sites

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/imperium-codebaker/connection-check/internal/geo"
)

// DefaultPort is the TCP port timed when pings are blocked and no port is given.
const DefaultPort = 443

// Site is one remote host.
type Site struct {
	Name  string `json:"name"`
	Host  string `json:"host"`
	Port  int    `json:"port"`
	Place string `json:"place,omitempty"` // airport code of where the host is, if the route can't show it
}

// Parse reads "Name=host[:port][@place]" or "host[:port][@place]", where place is an
// airport code such as "atl".
func Parse(spec string) (Site, error) {
	name, hostport, found := strings.Cut(spec, "=")
	if !found {
		hostport = name
	}
	hostport, place, err := cutPlace(hostport, spec)
	if err != nil {
		return Site{}, err
	}
	host, port := hostport, DefaultPort
	if h, p, err := net.SplitHostPort(hostport); err == nil {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return Site{}, fmt.Errorf("bad port in %q", spec)
		}
		host, port = h, n
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return Site{}, fmt.Errorf("no host in %q", spec)
	}
	name = strings.TrimSpace(name)
	if !found || name == "" {
		name = host
	}
	return Site{Name: name, Host: host, Port: port, Place: place}, nil
}

// cutPlace splits an "@place" airport code off hostport and checks that the code is known.
func cutPlace(hostport, spec string) (rest, place string, err error) {
	rest, place, found := strings.Cut(hostport, "@")
	if !found {
		return rest, "", nil
	}
	place = strings.ToLower(strings.TrimSpace(place))
	if geo.Airport(place) == nil {
		return "", "", fmt.Errorf("unknown airport code %q in %q", place, spec)
	}
	return rest, place, nil
}

// Path returns where the site list is saved for this user.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "connection-check", "sites.json"), nil
}

// Load reads the saved site list; a missing file gives an empty list.
func Load(path string) ([]Site, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var list []Site
	return list, json.Unmarshal(b, &list)
}

// Save writes the site list, creating its folder if needed.
func Save(path string, list []Site) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
