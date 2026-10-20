// Marks for the Over time chart: axes, guide lines, the series, lost pings, the blue
// window, route changes and the hover readout. Each builder takes the chart scale s.

import { formatClock, formatShortClock, formatShortRange, svgEl } from "./view.js";

const TICK_MINUTES = [1, 2, 5, 10, 15, 30, 60];
const LOSS_TICK_PX = 8;
const READOUT = { w: 236, h: 30, gap: 10 };

/** How each measure reads from a bucket, its unit, its lowest top value, and its guide lines. */
export const MEASURES = {
  avg: {
    value: (b) => (b.sent > b.lost ? b.avg : null), unit: "ms", floor: 120,
    guides: [{ v: 100, word: "fair" }, { v: 250, word: "poor" }],
  },
  loss: {
    value: (b) => (b.sent ? (100 * b.lost) / b.sent : null), unit: "%", floor: 5,
    guides: [{ v: 0.25, word: "fair" }, { v: 0.5, word: "poor" }],
  },
  jitter: {
    value: (b) => (b.sent > b.lost + 1 ? b.jitter : null), unit: "ms", floor: 40,
    guides: [{ v: 10, word: "fair" }, { v: 30, word: "poor" }],
  },
};

/** Return the top of the value axis for the buckets on show. */
export function valueMax(buckets, measure) {
  const m = MEASURES[measure];
  const top = Math.max(0, ...buckets.map((b) => m.value(b) || 0));
  const step = m.unit === "%" ? 1 : 50;
  return Math.max(m.floor, Math.ceil((top * 1.15) / step) * step);
}

function fmt(v, unit) {
  return unit === "%" ? `${+v.toFixed(2)}%` : `${Math.round(v)} ms`;
}

/** Return the axis line, guide lines and their labels. */
export function guides(s) {
  const m = MEASURES[s.measure];
  const out = [svgEl("line", { class: "axis", x1: s.x0, x2: s.x1, y1: s.y0, y2: s.y0 }),
    svgEl("text", { class: "ylabel", x: s.x0 - 6, y: s.y0 + 4, text: m.unit === "%" ? "0%" : "0 ms" })];
  for (const g of m.guides.filter((g) => g.v <= s.yMax)) {
    const y = s.y(g.v).toFixed(1);
    out.push(svgEl("line", { class: `guide guide-${g.word}`, x1: s.x0, x2: s.x1, y1: y, y2: y }),
      svgEl("text", { class: "ylabel", x: s.x0 - 6, y: +y + 4, text: fmt(g.v, m.unit) }),
      svgEl("text", { class: `glabel glabel-${g.word}`, x: s.x1 - 2, y: y - 4, text: g.word }));
  }
  return out;
}

/** Return the time ticks along the bottom. */
export function timeTicks(s) {
  const pxPerMinute = ((s.x1 - s.x0) * 60000) / (s.t1 - s.t0);
  const step = (TICK_MINUTES.find((m) => m * pxPerMinute >= 64) || 60) * 60000;
  const out = [];
  for (let t = Math.ceil(s.t0 / step) * step; t <= s.t1; t += step) {
    const x = s.x(t).toFixed(1);
    out.push(svgEl("line", { class: "tick", x1: x, x2: x, y1: s.y0, y2: s.y0 + 4 }),
      svgEl("text", { class: "xlabel", x, y: s.y0 + 16, text: formatShortClock(t) }));
  }
  return out;
}

/** Return the series: a line with an area under it, or bars for loss. */
export function series(buckets, s) {
  const m = MEASURES[s.measure];
  if (s.measure === "loss") {
    return buckets.filter((b) => m.value(b) > 0).map((b) => {
      const top = s.y(Math.min(m.value(b), s.yMax));
      return svgEl("rect", { class: "bar", x: s.x(b.t).toFixed(1), y: top.toFixed(1), width: s.slot, height: (s.y0 - top).toFixed(1) });
    });
  }
  const runs = [[]];
  for (const b of buckets) {
    const v = m.value(b);
    if (v === null || v <= 0) runs.push([]);
    else runs.at(-1).push([s.x(b.t + s.span / 2), s.y(Math.min(v, s.yMax))]);
  }
  let line = "";
  let area = "";
  for (const run of runs.filter((r) => r.length)) {
    const pts = run.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join("L");
    line += run.length > 1 ? `M${pts}` : `M${pts}h0.5`;
    area += `M${run[0][0].toFixed(1)},${s.y0}L${pts}L${run.at(-1)[0].toFixed(1)},${s.y0}Z`;
  }
  return [svgEl("path", { class: "area", d: area }), svgEl("path", { class: "line", d: line })];
}

/** Return red marks for lost pings: a short tick, or a full bar where every ping was lost. */
export function lostMarks(buckets, s) {
  if (s.measure === "loss") return [];
  return buckets.filter((b) => b.lost > 0).map((b) => {
    const all = b.lost >= b.sent;
    const h = all ? s.y0 - s.y1 : LOSS_TICK_PX;
    return svgEl("rect", { class: all ? "lost all" : "lost", x: s.x(b.t).toFixed(1), y: (s.y0 - h).toFixed(1), width: s.slot, height: h },
      svgEl("title", { text: `${b.lost} of ${b.sent} pings lost at ${formatClock(b.t)}` }));
  });
}

/** Return the blue window band with its edges and its from–to label. */
export function windowBand(s, from, to) {
  const a = Math.max(s.x0, s.x(from));
  const b = Math.min(s.x1, s.x(to));
  if (b <= a) return [];
  return [
    svgEl("rect", { class: "window", x: a.toFixed(1), y: s.y1, width: (b - a).toFixed(1), height: s.y0 - s.y1 }),
    svgEl("line", { class: "window-edge", x1: a, x2: a, y1: s.y1, y2: s.y0 }),
    svgEl("line", { class: "window-edge", x1: b, x2: b, y1: s.y1, y2: s.y0 }),
    windowLabel(s, a, b, formatShortRange(from, to)),
  ];
}

/** Label the band inside its left edge, or outside it when the band is too narrow. */
function windowLabel(s, a, b, text) {
  const inside = b - a >= text.length * 7.5 + 12;
  return svgEl("text", {
    class: inside ? "window-label" : "window-label outside", x: inside ? a + 6 : a - 6, y: s.y1 + 14, text,
  });
}

/** Return a band for a drag in progress. */
export function selection(s, from, to) {
  const a = Math.max(s.x0, s.x(Math.min(from, to)));
  const b = Math.min(s.x1, s.x(Math.max(from, to)));
  return svgEl("rect", { class: "selection", x: a.toFixed(1), y: s.y1, width: Math.max(0, b - a).toFixed(1), height: s.y0 - s.y1 });
}

/** Return a marker at each route change on show. */
export function changeMarks(changes, s) {
  return changes.filter((c) => c.t >= s.t0 && c.t <= s.t1).map((c) => {
    const x = s.x(c.t).toFixed(1);
    return svgEl("g", { class: "change" },
      svgEl("title", { text: `${formatClock(c.t)} · Route changed at hop ${c.ttl}` }),
      svgEl("line", { x1: x, x2: x, y1: s.y1, y2: s.y0 }),
      svgEl("path", { d: `M${x},${s.y1 - 2}l5,5l-5,5l-5,-5z` }),
      svgEl("rect", { class: "hit", x: +x - 6, y: s.y1, width: 12, height: s.y0 - s.y1 }));
  });
}

/** Return the hover readout for the bucket under time t: "{time} · {avg} ms · {lost}/{sent} lost". */
export function readout(buckets, s, t) {
  const b = buckets.find((x) => t >= x.t && t < x.t + s.span);
  if (!b) return [];
  const x = s.x(b.t + s.span / 2);
  const v = MEASURES[s.measure].value(b);
  const avg = b.sent > b.lost ? `${b.avg.toFixed(1)} ms` : "no reply";
  const text = `${formatClock(b.t)} · ${avg} · ${b.lost}/${b.sent} lost`;
  const left = x + READOUT.gap + READOUT.w > s.x1 ? x - READOUT.gap - READOUT.w : x + READOUT.gap;
  const top = s.y1 + 24;
  return [
    svgEl("line", { class: "hover-line", x1: x, x2: x, y1: s.y1, y2: s.y0 }),
    v !== null && svgEl("circle", { class: "hover-dot", cx: x, cy: s.y(Math.min(v, s.yMax)), r: 5 }),
    svgEl("rect", { class: "readout", x: left, y: top, width: READOUT.w, height: READOUT.h, rx: 6 }),
    svgEl("text", { class: "readout-text", x: left + 10, y: top + 20, text }),
  ];
}
