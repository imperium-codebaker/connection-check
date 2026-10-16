// Command connection-check traces the route to remote-desktop hosts and shows a live verdict in the browser.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/johncbrubaker/connection-check/internal/monitor"
	"github.com/johncbrubaker/connection-check/internal/probe"
	"github.com/johncbrubaker/connection-check/internal/server"
	"github.com/johncbrubaker/connection-check/internal/sites"
	"github.com/johncbrubaker/connection-check/web"
)

// siteFlags collects repeated --site values.
type siteFlags []string

func (f *siteFlags) String() string     { return strings.Join(*f, ", ") }
func (f *siteFlags) Set(v string) error { *f = append(*f, v); return nil }

type options struct {
	sites      siteFlags
	noBrowser  bool
	reportBase string
	runFor     time.Duration
	port       int
}

func main() {
	var o options
	flag.Var(&o.sites, "site", "site to check as 'Name=host[:port]'; repeatable; replaces the saved list")
	flag.BoolVar(&o.noBrowser, "no-browser", false, "don't open the browser")
	flag.StringVar(&o.reportBase, "report-base", "https://johncbrubaker.github.io/connection-check/", "report page URL")
	flag.DurationVar(&o.runFor, "for", 2*time.Hour, "stop after this long")
	flag.IntVar(&o.port, "port", 0, "local port for the page (0 picks a free one)")
	flag.Parse()
	if err := run(o); err != nil {
		log.Fatal(err)
	}
}

func run(o options) error {
	path, err := sites.Path()
	if err != nil {
		return err
	}
	list, err := siteList(o.sites, path)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, o.runFor)
	defer cancel()
	m, err := startMonitor(ctx, list)
	if err != nil {
		return err
	}
	return serve(ctx, m, o, path)
}

// siteList returns the sites given on the command line, saving them, or else the saved list.
func siteList(specs []string, path string) ([]sites.Site, error) {
	if len(specs) == 0 {
		return sites.Load(path)
	}
	var list []sites.Site
	for _, spec := range specs {
		s, err := sites.Parse(spec)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, sites.Save(path, list)
}

func startMonitor(ctx context.Context, list []sites.Site) (*monitor.Monitor, error) {
	sender, err := probe.Open()
	if err != nil {
		return nil, fmt.Errorf("can't send pings: %w", err)
	}
	go func() { <-ctx.Done(); sender.Close() }()
	if mode := sender.Mode(); mode.Partial {
		fmt.Println(mode.Hint)
	}
	m := monitor.New(ctx, sender)
	for _, s := range list {
		if err := m.Add(s); err != nil {
			fmt.Printf("Skipping %s: %v\n", s.Name, err)
		}
	}
	return m, nil
}

func serve(ctx context.Context, m *monitor.Monitor, o options, path string) error {
	ln, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", o.port))
	if err != nil {
		return err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	hosts := []string{fmt.Sprintf("127.0.0.1:%d", port), fmt.Sprintf("localhost:%d", port)}
	srv := &http.Server{Handler: server.New(m, server.Config{
		Page: web.FS, ReportBase: o.reportBase, SitesPath: path, Hosts: hosts, Started: time.Now(),
	})}
	url := "http://" + hosts[0] + "/"
	fmt.Printf("Connection Check is running at %s\nIt stops by itself after %s; press Ctrl+C to stop sooner.\n", url, o.runFor)
	if !o.noBrowser {
		openBrowser(url)
	}
	go func() { <-ctx.Done(); srv.Close() }()
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	fmt.Println("Stopped.")
	return nil
}
