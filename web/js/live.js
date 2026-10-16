// Talks to the local Connection Check program: the event stream and its small JSON API.

const STALE_MS = 5000;

/** Open the event stream; onPayload gets each payload, onAlive(bool) fires when liveness changes. */
export function connectStream(onPayload, onAlive) {
  const conn = { last: Date.now(), alive: null };
  const source = new EventSource("api/stream");
  source.addEventListener("state", (event) => receive(conn, event, onPayload, onAlive));
  source.addEventListener("error", () => setAlive(conn, false, onAlive));
  setInterval(() => {
    if (Date.now() - conn.last > STALE_MS) setAlive(conn, false, onAlive);
  }, 1000);
}

function receive(conn, event, onPayload, onAlive) {
  let payload;
  try {
    payload = JSON.parse(event.data);
  } catch {
    return;
  }
  conn.last = Date.now();
  setAlive(conn, true, onAlive);
  onPayload(payload);
}

function setAlive(conn, alive, onAlive) {
  if (conn.alive === alive) return;
  conn.alive = alive;
  onAlive(alive);
}

async function send(method, path, body) {
  const init = { method, headers: { Accept: "application/json" } };
  if (body) {
    init.headers["Content-Type"] = "application/json";
    init.body = JSON.stringify(body);
  }
  const res = await fetch(path, init);
  if (!res.ok) throw new Error((await res.text()).trim() || `the program answered ${res.status}`);
  return res;
}

/** Ask the program to show a window: "default", "bad", or "custom" with {from, to}. */
export function postWindow(site, window, range = {}) {
  return send("POST", "api/window", { site, window, ...range });
}

/** Fetch the report link and text for a site. */
export async function fetchReport(site) {
  const res = await send("GET", `api/report?site=${encodeURIComponent(site)}`);
  return res.json();
}

/** Add the first site to check. */
export function postSite(name, host) {
  return send("POST", "api/sites", { name, host });
}
