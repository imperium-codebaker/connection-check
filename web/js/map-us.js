// Across the country: one dot per city on the US outline, joined in route order.

import { cityStops, label, loadOutline, pulseRing, rowKeys, travelDot, travelSeconds } from "./map-geo.js";
import { routerName } from "./model.js";
import { formatMs, gradeWord, reducedMotion, svgEl } from "./view.js";

const FRAME = { x: 0, y: 0, w: 975, h: 610 };
const CURVE_MIN_PX = 150;
const CURVE_BEND = 0.22;
const NEAR_BOX = { w: 72, h: 45 };

/** Create the US map on an SVG element. */
export function createUsMap(svg) {
  const map = { svg, key: "", linkLabel: null };
  const nation = svgEl("path", { class: "nation" });
  const states = svgEl("path", { class: "states" });
  map.route = svgEl("g", { class: "route" });
  map.motion = svgEl("g", { class: "motion", "aria-hidden": "true" });
  svg.replaceChildren(nation, states, map.route, map.motion);
  loadOutline(svg, nation, states);
  return {
    /** Draw snap; opts.live adds motion, opts.near outlines the first city. Returns false when nothing is placed. */
    render(snap, opts) {
      const stops = cityStops(snap);
      if (stops.length) draw(map, snap, stops, opts);
      return stops.length > 0;
    },
  };
}

/** Return how much to enlarge marks and labels so they stay legible on a narrow map. */
function enlarge(svg) {
  return Math.round(Math.max(1, 640 / (svg.clientWidth || FRAME.w)) * 4) / 4;
}

function draw(map, snap, stops, opts) {
  opts = { ...opts, k: enlarge(map.svg) };
  const link = snap.stretches?.find((s) => s.kind === "link") || null;
  const linked = Boolean(link && stops.length > 1 && stops.at(-1).dest);
  const key = JSON.stringify([stops.map((s) => [s.city, s.grade, s.approx, s.dest]), linked, opts]);
  if (key !== map.key) {
    map.key = key;
    map.route.replaceChildren(...marks(map, snap, stops, { ...opts, linked }));
    map.motion.replaceChildren(...motion(map, stops, link, { ...opts, linked }));
  }
  if (map.linkLabel) map.linkLabel.textContent = `Private link · +${Math.round(link.ms)} ms`;
}

function segmentPath(a, b) {
  const len = Math.hypot(b.x - a.x, b.y - a.y);
  if (len < CURVE_MIN_PX) return `M${a.x.toFixed(1)},${a.y.toFixed(1)}L${b.x.toFixed(1)},${b.y.toFixed(1)}`;
  const cx = (a.x + b.x) / 2;
  const cy = Math.min(a.y, b.y) - len * CURVE_BEND;
  return `M${a.x.toFixed(1)},${a.y.toFixed(1)}Q${cx.toFixed(1)},${cy.toFixed(1)} ${b.x.toFixed(1)},${b.y.toFixed(1)}`;
}

function segId(map, i) {
  return `${map.svg.id}-seg-${i}`;
}

function marks(map, snap, stops, opts) {
  const out = [];
  map.linkLabel = null;
  if (opts.near) out.push(nearBox(stops.find((s) => !s.dest) || stops[0]));
  for (let i = 1; i < stops.length; i++) {
    const isLink = opts.linked && i === stops.length - 1;
    const d = segmentPath(stops[i - 1], stops[i]);
    if (isLink) out.push(svgEl("path", { class: "seg-halo", d }));
    out.push(svgEl("path", { id: segId(map, i), class: isLink ? "seg link" : `seg g-${stops[i].grade}`, d }));
  }
  out.push(...stops.map((stop) => node(stop, snap.site, opts.k)));
  out.push(...cityLabels(stops, opts.k));
  if (opts.linked) out.push(linkLabel(map, stops.at(-2), stops.at(-1), opts.k));
  return out;
}

function nearBox(stop) {
  return svgEl("rect", {
    class: "near-box", x: stop.x - NEAR_BOX.w / 2, y: stop.y - NEAR_BOX.h / 2,
    width: NEAR_BOX.w, height: NEAR_BOX.h, rx: 4,
  });
}

function node(stop, site, k) {
  const classes = ["node", `g-${stop.grade}`, stop.approx && "approx", stop.dest && "dest"];
  return svgEl("circle", {
    class: classes.filter(Boolean).join(" "), cx: stop.x, cy: stop.y, r: (stop.dest ? 12 : 8) * k,
    "data-keys": rowKeys(stop.hops),
  }, svgEl("title", { text: nodeTitle(stop, site) }));
}

function nodeTitle(stop, site) {
  const head = stop.approx ? `${stop.city} (approximate)` : stop.city;
  const lines = stop.hops.map((hop) => {
    const who = hop.ttl === 0 ? site : `#${hop.ttl} ${routerName(hop)}`;
    return hop.answered ? `${who}: ${gradeWord(hop.grade)}, ${formatMs(hop.avg)}` : `${who}: no reply`;
  });
  return [head, ...lines].join("\n");
}

function cityLabels(stops, k) {
  const seen = new Set();
  const out = [];
  for (const stop of [...stops].reverse()) {
    if (seen.has(stop.city)) continue;
    seen.add(stop.city);
    const size = (stop.dest ? 20 : 17) * k;
    out.push(label(stop.x, stop.dest ? stop.y + 16 * k + size : stop.y - 12 * k, stop.city, size, FRAME));
  }
  return out;
}

function linkLabel(map, a, b, k) {
  const len = Math.hypot(b.x - a.x, b.y - a.y);
  const y = len < CURVE_MIN_PX ? (a.y + b.y) / 2 - 14 : Math.min(a.y, b.y) - len * CURVE_BEND * 0.5 - 8;
  map.linkLabel = label((a.x + b.x) / 2, y, "", 17 * k, FRAME);
  return map.linkLabel;
}

function motion(map, stops, link, opts) {
  if (!opts.live || reducedMotion()) return [];
  const out = [];
  for (let i = 1; i < stops.length; i++) {
    const isLink = opts.linked && i === stops.length - 1;
    const ms = isLink ? link.ms : stops[i].ms - stops[i - 1].ms;
    out.push(travelDot(segId(map, i), travelSeconds(ms), isLink ? 4 : 3, isLink ? "dark" : "light"));
  }
  const dest = stops.find((s) => s.dest);
  if (dest) out.unshift(pulseRing(dest.x, dest.y, 12 * opts.k));
  return out;
}
