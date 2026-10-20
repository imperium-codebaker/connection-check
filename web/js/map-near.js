// Near you: a schematic chain from home through every stretch that ends close to the user.

import { label, pulseRing, rowKeys, travelDot, travelSeconds } from "./map-geo.js";
import { nearCity, siteCity, stretchHops, stretchTitle } from "./model.js";
import { reducedMotion, svgEl } from "./view.js";

const FRAME = { x: 0, y: 0, w: 640, h: 400 };
const HOME = { x: 560, y: 96 };
const CHAIN_END = { x: 262, y: 276 };
const STUB_END = { x: 640, y: 330 };

/** Create the near-you map on an SVG element; title is its caption element. */
export function createNearMap(svg, title) {
  const map = { svg, title, key: "" };
  map.route = svgEl("g", { class: "route" });
  map.motion = svgEl("g", { class: "motion", "aria-hidden": "true" });
  svg.replaceChildren(map.route, map.motion);
  return {
    /** Draw snap; opts.live adds motion. Returns false when no stretch before the site has a place. */
    render(snap, opts) {
      const city = nearCity(snap);
      if (city) draw(map, snap, city, opts);
      return Boolean(city);
    },
  };
}

function draw(map, snap, city, opts) {
  opts = { ...opts, k: Math.round(Math.max(1, 420 / (map.svg.clientWidth || FRAME.w)) * 4) / 4 };
  const plan = layout(snap, city);
  const key = JSON.stringify([city, plan.dots.map((d) => [d.title, d.grade, d.site]), plan.stub, opts]);
  if (key === map.key) return;
  map.key = key;
  map.title.textContent = `Near you: ${city}`;
  map.svg.setAttribute("aria-label", `Route near ${city}`);
  map.route.replaceChildren(...marks(map, plan, opts.k));
  map.motion.replaceChildren(...motion(map, plan, opts));
}

/** Place home, a dot per stretch in the city, and a stub toward the rest of the route. */
function layout(snap, city) {
  const list = snap.stretches;
  const nearby = list.filter((s) => s.near && s.kind !== "home" && s.kind !== "link");
  const dots = nearby.map((s, i) => {
    const t = (i + 1) / nearby.length;
    return {
      x: HOME.x + (CHAIN_END.x - HOME.x) * t, y: HOME.y + (CHAIN_END.y - HOME.y) * t,
      title: stretchTitle(s, snap.site), grade: s.grade || "measuring", site: s.kind === "site",
      ms: s.ms, keys: rowKeys(stretchHops(snap, s)),
    };
  });
  const last = nearby.at(-1);
  const after = list[list.indexOf(last) + 1];
  const stub = after && !dots.at(-1).site ? {
    link: after.kind === "link",
    grade: after.grade || "measuring",
    ms: after.kind === "link" ? after.ms : list.at(-1).ms - last.ms,
    text: `to ${siteCity(snap) || snap.site}`,
  } : null;
  return { dots, stub };
}

function chainPath(dots) {
  return [HOME, ...dots].map((p, i) => `${i ? "L" : "M"}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join("");
}

function stubPath(from) {
  const c1 = { x: from.x + 78, y: from.y + 54 };
  const c2 = { x: from.x + 218, y: from.y + 64 };
  return `M${from.x.toFixed(1)},${from.y.toFixed(1)}C${c1.x},${c1.y} ${c2.x},${c2.y} ${STUB_END.x},${STUB_END.y}`;
}

function marks(map, { dots, stub }, k) {
  const out = [svgEl("path", { id: `${map.svg.id}-chain`, class: `seg g-${dots.at(-1).grade}`, d: chainPath(dots) })];
  if (stub) {
    const d = stubPath(dots.at(-1));
    if (stub.link) out.push(svgEl("path", { class: "seg-halo", d }));
    out.push(svgEl("path", { id: `${map.svg.id}-stub`, class: stub.link ? "seg link" : `seg g-${stub.grade}`, d }));
  }
  out.push(svgEl("rect", { class: "home", x: HOME.x - 11, y: HOME.y - 11, width: 22, height: 22, rx: 4 }));
  out.push(label(HOME.x + 20, HOME.y + 20 + 17 * k, "Home", 17 * k, FRAME));
  dots.forEach((dot, i) => out.push(...dotMarks(dot, i === dots.length - 1, k)));
  if (stub) out.push(label(STUB_END.x - 60, STUB_END.y - 16, stub.text, 17 * k, FRAME, "maplabel quiet"));
  return out;
}

function dotMarks(dot, last, k) {
  const classes = ["node", `g-${dot.grade}`, dot.site && "dest"];
  const circle = svgEl("circle", {
    class: classes.filter(Boolean).join(" "), cx: dot.x, cy: dot.y, r: last ? 13 : 11, "data-keys": dot.keys,
  }, svgEl("title", { text: dot.title }));
  const text = last
    ? svgEl("text", { class: "maplabel end", x: dot.x - 18, y: dot.y - 14, "font-size": 17 * k, text: dot.title })
    : svgEl("text", { class: "maplabel start", x: dot.x + 18, y: dot.y - 8, "font-size": 17 * k, text: dot.title });
  return [circle, text];
}

function motion(map, { dots, stub }, opts) {
  if (!opts.live || reducedMotion()) return [];
  const out = [travelDot(`${map.svg.id}-chain`, travelSeconds(dots.at(-1).ms), 3, "light")];
  if (stub) out.push(travelDot(`${map.svg.id}-stub`, travelSeconds(stub.ms), stub.link ? 4 : 3, "dark"));
  const site = dots.find((d) => d.site);
  if (site) out.unshift(pulseRing(site.x, site.y, 13));
  return out;
}
