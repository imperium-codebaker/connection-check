// Shared map helpers: projection, the US outline, route stops by city, labels and moving dots.

import { rowKey, worstGrade } from "./model.js";
import { svgEl } from "./view.js";

const MIN_TRAVEL_S = 0.8;
const MAX_TRAVEL_S = 6;
const MS_PER_TRAVEL_S = 10;

let projection = null;
let outline = null;

/** Project a place to US map coordinates (975 × 610), or null. */
export function project(place) {
  if (!place || !window.d3?.geoAlbersUsa) return null;
  projection ??= window.d3.geoAlbersUsa().scale(1300).translate([487.5, 305]);
  const p = projection([place.lon, place.lat]);
  return p ? { x: p[0], y: p[1] } : null;
}

/** Fill the nation and state paths from the bundled outline; adds "no-outline" on failure. */
export async function loadOutline(svg, nation, states) {
  try {
    outline ??= fetch("vendor/us-outline.json").then((res) => res.json());
    const shapes = await outline;
    nation.setAttribute("d", shapes.nation);
    states.setAttribute("d", shapes.states);
  } catch {
    svg.classList.add("no-outline");
  }
}

/**
 * Return the placed stops along the route, consecutive hops in one city merged:
 * [{city, approx, x, y, hops, dest, grade, ms}]. ms is the round trip at the stop,
 * capped at every later stop's so a slow-answering router doesn't run time backwards.
 */
export function cityStops(snap) {
  const stops = [];
  for (const hop of [...snap.hops, snap.dest]) {
    const pt = project(hop.place);
    if (!pt) continue;
    const last = stops.at(-1);
    if (last && last.city === hop.place.city) last.hops.push(hop);
    else stops.push({ city: hop.place.city, x: pt.x, y: pt.y, hops: [hop] });
  }
  let low = Infinity;
  for (let i = stops.length - 1; i >= 0; i--) {
    const stop = stops[i];
    stop.dest = stop.hops.includes(snap.dest);
    stop.approx = stop.hops.every((h) => h.place.approx);
    stop.grade = worstGrade(stop.hops);
    for (const h of stop.hops) if (h.answered && h.avg > 0) low = Math.min(low, h.avg);
    stop.ms = Number.isFinite(low) ? low : 0;
  }
  return stops;
}

/** Return the seconds a moving dot takes over a stretch that adds ms of delay. */
export function travelSeconds(ms) {
  return Math.min(MAX_TRAVEL_S, Math.max(MIN_TRAVEL_S, MIN_TRAVEL_S + ms / MS_PER_TRAVEL_S));
}

/** Return a dot that travels along the path with the given id, taking seconds per trip. */
export function travelDot(pathId, seconds, r, cls) {
  return svgEl("circle", { class: `traveller ${cls}`, r },
    svgEl("animateMotion", { dur: `${seconds.toFixed(1)}s`, repeatCount: "indefinite", rotate: "auto" },
      svgEl("mpath", { href: `#${pathId}` })));
}

/** Return a text label with a halo, kept inside the frame {x, y, w, h}. */
export function label(x, y, text, size, frame, cls = "maplabel") {
  const half = (text.length * size * 0.3) + 4;
  const cx = Math.min(frame.x + frame.w - half, Math.max(frame.x + half, x));
  return svgEl("text", { class: cls, x: cx, y, "font-size": size, text });
}

/** Return a ring that pulses around a point. */
export function pulseRing(x, y, r) {
  return svgEl("circle", { class: "pulse-ring", cx: x, cy: y, r });
}

/** Return the data-keys value naming the table rows a map mark stands for. */
export function rowKeys(hops) {
  return hops.map(rowKey).join(" ");
}
