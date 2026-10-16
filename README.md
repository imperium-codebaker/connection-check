# Connection Check

Finds out who owns a bad remote-desktop connection: your Wi-Fi or router, your
internet provider, the carriers in between, or the remote site.

It traces the route to the remote host, pings every router on the way once a
second and the host itself five times a second, and shows a live view in your
browser:

- a plain-English verdict and what to do about it;
- every hop with its owner, loss, latency and jitter;
- a route map;
- a latency timeline you can drag to pick a time range;
- an "It's bad right now" button that focuses the verdict on that moment;
- "Copy report for IT": a link that opens the same view anywhere, plus a
  plain-text summary.

Grades follow HP Anyware's connection health scale: latency is Fair above 50 ms
and Poor above 100 ms; loss is Fair above 0.25% and Poor above 0.5%. As in
PingPlotter, only the destination decides the grade. A router in the middle that
drops or delays pings counts only if the trouble carries through to the
destination; otherwise it is just a router that ignores pings.

## Running it

Whoever supports your remote desktop gives you one line to paste. It downloads
the program for your computer to your own app folder and starts it; nothing is
installed system-wide and no administrator rights are needed.

macOS and Linux (Terminal):

```sh
curl -fsSL https://github.com/johncbrubaker/connection-check/releases/latest/download/install.sh \
  | sh -s -- --site 'Office=remote.example.com'
```

Windows (PowerShell):

```powershell
& ([scriptblock]::Create((irm https://github.com/johncbrubaker/connection-check/releases/latest/download/install.ps1))) -Site 'Office=remote.example.com'
```

`--site` takes `Name=host[:port]` and can be repeated. The list is saved, so
later runs need no arguments. The program stops after two hours, or when you
press Ctrl+C or close the window.

On Linux distributions that stop ordinary users sending pings (Ubuntu, Debian,
Pop!_OS), the installer offers to allow them once, as Fedora does by default.
Without that, hops are found with UDP probes and the host is timed with TCP
connects, and some distant routers won't show.

## What it does not see

Problems that don't affect pings: a remote-desktop stream being throttled by
port, traffic split across parallel links, or a host short of CPU.

## Report links

The report is compressed into the part of the link after `#`, which browsers
never send to a server. The page at
<https://johncbrubaker.github.io/connection-check/> decodes it in the browser.

## Building

```sh
go test ./...
go build ./cmd/connection-check
```

Tagging `v*` builds macOS, Linux and Windows binaries and publishes a release.

Map data: US outline from [us-atlas](https://github.com/topojson/us-atlas),
pre-rendered by `tools/make-us-outline.cjs`; airport locations from
[OurAirports](https://ourairports.com/data/) (public domain).
