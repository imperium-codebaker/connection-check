// Motion that shows something real happening: routers fading in as the trace finds them,
// and a red ripple on a router in the second it loses a ping. Off in reports and under
// reduced motion; map animations pause while the tab is hidden.

import { rowKey } from "./model.js";
import { reducedMotion, svgEl } from "./view.js";

const RIPPLE_MS = 1000;
const NONE = { fresh: new Set(), lost: new Set() };

/** Create the motion tracker; svgs are the map elements whose animations pause with the tab. */
export function createMotion(svgs) {
  const sites = new Map();
  document.addEventListener("visibilitychange", () => {
    for (const svg of svgs) {
      if (document.hidden) svg.pauseAnimations();
      else svg.unpauseAnimations();
    }
  });
  return {
    /** Return the row keys to fade in and to ripple for this update of site index. */
    marks(index, snap, live) {
      const seen = sites.get(index) || { keys: new Set(), counts: new Map(), window: "" };
      sites.set(index, seen);
      const marks = { fresh: freshKeys(seen, snap), lost: lostKeys(seen, snap) };
      return live && !reducedMotion() ? marks : NONE;
    },
    /** Ripple every map mark that stands for one of keys. */
    ripple(keys) {
      if (!keys.size) return;
      for (const svg of svgs) rippleMarks(svg, keys);
    },
  };
}

function rows(snap) {
  return [...snap.hops.filter((h) => h.answered || h.addr), snap.dest];
}

function freshKeys(seen, snap) {
  const fresh = new Set();
  for (const hop of rows(snap)) {
    const key = rowKey(hop);
    if (hop.answered && !seen.keys.has(key)) fresh.add(key);
    if (hop.answered) seen.keys.add(key);
  }
  return fresh;
}

/** Keys of rows that lost a ping since the last update of the same window. */
function lostKeys(seen, snap) {
  const lost = new Set();
  const kind = snap.window === "all" ? `all|${Math.round((snap.to - snap.from) / 60000)}` : snap.window;
  const sameWindow = seen.window === kind;
  seen.window = kind;
  for (const hop of rows(snap)) {
    const key = rowKey(hop);
    const before = seen.counts.get(key);
    if (sameWindow && before && !hop.ignores && hop.lost > before.lost && hop.sent > before.sent) lost.add(key);
    seen.counts.set(key, { sent: hop.sent, lost: hop.lost });
  }
  return lost;
}

function rippleMarks(svg, keys) {
  for (const mark of svg.querySelectorAll("[data-keys]")) {
    if (!mark.dataset.keys.split(" ").some((k) => keys.has(k))) continue;
    const ring = svgEl("circle", {
      class: "ripple-ring", cx: mark.getAttribute("cx"), cy: mark.getAttribute("cy"), r: mark.getAttribute("r"),
    });
    mark.after(ring);
    setTimeout(() => ring.remove(), RIPPLE_MS);
  }
}
