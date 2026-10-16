// Route map: the US outline with each located hop, plus a strip for hops with no location.

import { routerName, worstGrade } from "./model.js";
import { el, formatMs, gradeWord, svgEl } from "./view.js";

const SIZE = { w: 975, h: 610 };
const ZOOM_KM = 150;
const MIN_ZOOM_W = 40;
const HOME_OFFSET_PX = 16;

let projection = null;

/** Create the route map on an SVG element; strip is the list for hops without a place. */
export function createMap(svg, strip) {
  const map = { svg, strip, snap: null, key: "" };
  map.nation = svgEl("path", { class: "nation" });
  map.states = svgEl("path", { class: "states" });
  map.route = svgEl("g", { class: "route" });
  svg.replaceChildren(map.nation, map.states, map.route);
  loadOutline(map);
  new ResizeObserver(() => draw(map)).observe(svg);
  return {
    render(snap, flowing) {
      map.snap = snap;
      svg.classList.toggle("flowing", flowing);
      draw(map);
    },
  };
}

async function loadOutline(map) {
  try {
    const res = await fetch("vendor/us-outline.json");
    const outline = await res.json();
    map.nation.setAttribute("d", outline.nation);
    map.states.setAttribute("d", outline.states);
  } catch {
    map.svg.classList.add("no-outline");
  }
}

function project(place) {
  if (!place || !window.d3?.geoAlbersUsa) return null;
  projection ??= window.d3.geoAlbersUsa().scale(1300).translate([487.5, 305]);
  const p = projection([place.lon, place.lat]);
  return p ? { x: p[0], y: p[1] } : null;
}

function draw(map) {
  const width = map.svg.clientWidth;
  if (!map.snap || !width) return;
  const stops = routeStops(map.snap);
  const nodes = placedNodes(stops);
  const key = JSON.stringify([width, map.snap.site, nodes.map(nodeKey), stops.map(stopKey)]);
  if (key === map.key) return;
  map.key = key;
  const box = viewBox(nodes);
  const unit = box.w / width;
  map.svg.setAttribute("viewBox", [box.x, box.y, box.w, box.h].map((n) => n.toFixed(1)).join(" "));
  map.route.replaceChildren(...routeMarks(map.snap, nodes, { unit, box }));
  renderStrip(map.strip, map.snap, stops);
}

function routeStops(snap) {
  const stops = snap.hops.map((hop) => ({ hop, dest: false, pt: project(hop.place) }));
  stops.push({ hop: snap.dest, dest: true, pt: project(snap.dest.place) });
  return stops;
}

function placedNodes(stops) {
  const nodes = [];
  for (const stop of stops.filter((s) => s.pt)) {
    const last = nodes.at(-1);
    if (last && Math.hypot(last.x - stop.pt.x, last.y - stop.pt.y) < 0.5) last.stops.push(stop);
    else nodes.push({ x: stop.pt.x, y: stop.pt.y, stops: [stop] });
  }
  return nodes.map(describeNode);
}

function describeNode(node) {
  return {
    ...node,
    grade: worstGrade(node.stops.map((s) => s.hop)),
    dest: node.stops.some((s) => s.dest),
    approx: node.stops.every((s) => s.hop.place.approx),
    city: node.stops[0].hop.place.city,
  };
}

function nodeKey(n) {
  return `${n.x.toFixed(1)},${n.y.toFixed(1)},${n.grade},${n.approx},${n.stops.length}`;
}

function stopKey(s) {
  return `${s.hop.ttl}:${Boolean(s.pt)}:${s.hop.grade}:${s.hop.answered}:${routerName(s.hop)}`;
}

function viewBox(nodes) {
  if (!nodes.length || spreadKm(nodes) > ZOOM_KM) return { x: 0, y: 0, ...SIZE };
  const xs = nodes.map((n) => n.x);
  const ys = nodes.map((n) => n.y);
  const [bw, bh] = [Math.max(...xs) - Math.min(...xs), Math.max(...ys) - Math.min(...ys)];
  let w = Math.max(MIN_ZOOM_W, bw * 1.4 + 10);
  let h = (w * SIZE.h) / SIZE.w;
  if (h < bh * 1.4 + 10) [h, w] = [bh * 1.4 + 10, ((bh * 1.4 + 10) * SIZE.w) / SIZE.h];
  const cx = (Math.max(...xs) + Math.min(...xs)) / 2;
  const cy = (Math.max(...ys) + Math.min(...ys)) / 2;
  return { x: cx - w / 2, y: cy - h / 2, w, h };
}

function spreadKm(nodes) {
  let max = 0;
  for (const a of nodes) {
    for (const b of nodes) max = Math.max(max, distanceKm(a.stops[0].hop.place, b.stops[0].hop.place));
  }
  return max;
}

function distanceKm(a, b) {
  const rad = Math.PI / 180;
  const dLat = (b.lat - a.lat) * rad;
  const dLon = (b.lon - a.lon) * rad;
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(a.lat * rad) * Math.cos(b.lat * rad) * Math.sin(dLon / 2) ** 2;
  return 2 * 6371 * Math.asin(Math.sqrt(h));
}

function homePoint(nodes, { unit, box }) {
  const [first, next] = nodes;
  let [dx, dy] = [-1, 0.6];
  if (next && (next.x !== first.x || next.y !== first.y)) [dx, dy] = [first.x - next.x, first.y - next.y];
  const len = Math.hypot(dx, dy);
  const off = HOME_OFFSET_PX * unit;
  const margin = 10 * unit;
  const x = first.x + (dx / len) * off;
  const y = first.y + Math.abs(dy / len) * off;
  return {
    x: Math.min(box.x + box.w - margin, Math.max(box.x + margin, x)),
    y: Math.min(box.y + box.h - margin, Math.max(box.y + margin, y)),
  };
}

function routeMarks(snap, nodes, frame) {
  const { unit } = frame;
  if (!nodes.length) return [];
  const home = homePoint(nodes, frame);
  const points = [home, ...nodes];
  const segments = nodes.map((node, i) => segment(points[i], node, node.grade, unit));
  return [
    ...segments,
    homeMark(home, unit),
    ...nodes.map((node) => nodeMark(node, snap.site, unit)),
    ...labels(nodes, snap.site, frame),
  ];
}

function segment(a, b, grade, unit) {
  const screenLen = Math.max(1, Math.round(Math.hypot(b.x - a.x, b.y - a.y) / unit));
  return svgEl("line", {
    class: `seg g-${grade}`, x1: a.x, y1: a.y, x2: b.x, y2: b.y,
    pathLength: screenLen, "stroke-width": 2.5 * unit,
  });
}

function homeMark(home, unit) {
  const size = 11 * unit;
  return svgEl("rect", {
    class: "home", x: home.x - size / 2, y: home.y - size / 2, width: size, height: size,
    rx: 2 * unit, "stroke-width": 2 * unit,
  }, svgEl("title", { text: "Home: your computer and router" }));
}

function nodeMark(node, site, unit) {
  const classes = ["node", `g-${node.grade}`, node.approx && "approx", node.dest && "dest"];
  return svgEl("circle", {
    class: classes.filter(Boolean).join(" "), cx: node.x, cy: node.y,
    r: (node.dest ? 9 : 6) * unit, "stroke-width": 2 * unit,
    "stroke-dasharray": node.approx ? `${2.5 * unit} ${2 * unit}` : null,
  }, svgEl("title", { text: nodeTitle(node, site) }));
}

function nodeTitle(node, site) {
  const head = node.approx ? `${node.city} (approximate)` : node.city;
  const lines = node.stops.map(({ hop, dest }) => {
    const who = dest ? `${site} (destination)` : `#${hop.ttl} ${routerName(hop)}`;
    return hop.answered ? `${who}: ${gradeWord(hop.grade)}, ${formatMs(hop.avg)}` : `${who}: no reply`;
  });
  return [head, ...lines].join("\n");
}

function labels(nodes, site, frame) {
  const byPlace = new Map();
  for (const node of nodes) {
    const key = `${Math.round(node.x)},${Math.round(node.y)}`;
    if (!byPlace.has(key) || node.dest) byPlace.set(key, node);
  }
  const seen = new Set();
  const marks = [];
  for (const node of byPlace.values()) {
    const text = node.dest ? site : node.city;
    if (seen.has(text)) continue;
    seen.add(text);
    marks.push(label(node.x, node.y - (node.dest ? 14 : 10) * frame.unit, text, frame));
  }
  return marks;
}

function labelHalfWidth(text, unit) {
  return (text.length * 3.6 + 4) * unit;
}

function label(px, py, text, { unit, box }) {
  const half = labelHalfWidth(text, unit);
  const x = Math.min(box.x + box.w - half, Math.max(box.x + half, px));
  return svgEl("text", {
    class: "maplabel", x, y: py,
    "font-size": 12 * unit, "stroke-width": 3 * unit, text,
  });
}

function renderStrip(strip, snap, stops) {
  const box = strip.parentElement;
  box.hidden = stops.every((s) => s.pt);
  if (box.hidden) return strip.replaceChildren();
  const items = [anchor("Home")];
  let lastCity = null;
  for (const stop of stops) {
    const city = stop.pt ? stop.hop.place.city : null;
    if (city && city !== lastCity) items.push(anchor(city));
    if (!city) items.push(stripStop(stop, snap.site));
    lastCity = city;
  }
  strip.replaceChildren(...items);
}

function anchor(text) {
  return el("li", { class: "anchor" },
    el("span", { class: "pill" }, el("span", { class: "pin", "aria-hidden": "true" }), text));
}

function stripStop({ hop, dest }, site) {
  const grade = hop.answered ? hop.grade || "measuring" : "silent";
  const who = dest ? `${site}'s end` : `#${hop.ttl}`;
  const what = hop.answered ? routerName(hop) : "doesn't answer";
  const where = hop.place ? ` · ${hop.place.city}` : "";
  const state = hop.answered ? gradeWord(hop.grade) : "no reply";
  return el("li", { class: `stop g-${grade}`, title: `${who} ${what}${where}: ${state}` },
    el("span", { class: "pill" },
      el("span", { class: "dot", "aria-hidden": "true" }),
      el("strong", { text: who }), ` ${what}${where}`));
}
