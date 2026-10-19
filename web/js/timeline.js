// Destination latency timeline with loss ticks and drag-to-select.

import { formatClock, svgEl } from "./view.js";

const MARGIN = { left: 52, right: 12, top: 12, bottom: 24 };
const GUIDES = [{ ms: 100, word: "fair" }, { ms: 250, word: "poor" }];
const TICK_MINUTES = [1, 2, 5, 10, 15, 30, 60];
const MIN_DRAG_PX = 6;
const PENDING_MS = 4000;

/** Create a timeline on an SVG element; onSelect(from, to) gets drag selections in Unix ms. */
export function createTimeline(svg, onSelect) {
  const chart = { svg, onSelect, snap: null, now: 0, interactive: false, drag: null, pending: null };
  attachDrag(chart);
  new ResizeObserver(() => draw(chart)).observe(svg);
  return {
    render(snap, now, interactive) {
      Object.assign(chart, { snap, now, interactive: interactive && Boolean(onSelect) });
      expirePending(chart);
      draw(chart);
    },
  };
}

function draw(chart) {
  if (!chart.snap || !chart.svg.clientWidth) return;
  const s = (chart.scale = makeScale(chart));
  chart.svg.setAttribute("viewBox", `0 0 ${s.width} ${s.height}`);
  chart.svg.classList.toggle("selectable", chart.interactive);
  const { line, area } = seriesPaths(chart.snap.timeline, s);
  chart.svg.replaceChildren(
    band(s, chart.snap.from, chart.snap.to, "window"),
    ...yGuides(s), ...xTicks(s),
    svgEl("path", { class: "area", d: area }),
    svgEl("path", { class: "line", d: line }),
    ...lossTicks(chart.snap.timeline, s),
    selectionBand(chart, s));
}

function makeScale(chart) {
  const width = chart.svg.clientWidth;
  const height = chart.svg.clientHeight || 180;
  const [t0, t1, span] = timeDomain(chart);
  const yMax = valueMax(chart.snap.timeline);
  const x0 = MARGIN.left, x1 = width - MARGIN.right;
  const y0 = height - MARGIN.bottom, y1 = MARGIN.top;
  return {
    width, height, t0, t1, span, yMax, x0, x1, y0, y1,
    x: (t) => x0 + ((t - t0) / (t1 - t0)) * (x1 - x0),
    y: (v) => y0 - (v / yMax) * (y0 - y1),
    t: (px) => t0 + ((px - x0) / (x1 - x0)) * (t1 - t0),
  };
}

function timeDomain(chart) {
  const { timeline, from, to } = chart.snap;
  const span = timeline.length > 1 ? timeline[1].t - timeline[0].t : 10000;
  const t0 = timeline.length ? timeline[0].t : Math.min(from, chart.now - 15 * 60000);
  const end = timeline.length ? timeline.at(-1).t + span : chart.now;
  return [t0, Math.max(end, chart.now, to, t0 + span), span];
}

function valueMax(timeline) {
  const top = Math.max(0, ...timeline.map((b) => b.avg || 0));
  return Math.max(120, Math.ceil((top * 1.15) / 50) * 50);
}

function seriesPaths(timeline, s) {
  let line = "";
  let area = "";
  let run = [];
  const flush = () => {
    if (!run.length) return;
    const pts = run.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join("L");
    line += run.length > 1 ? `M${pts}` : `M${pts}h0.5`;
    area += `M${run[0][0].toFixed(1)},${s.y0}L${pts}L${run.at(-1)[0].toFixed(1)},${s.y0}Z`;
    run = [];
  };
  for (const b of timeline) {
    if (b.sent > 0 && b.avg > 0) run.push([s.x(b.t + s.span / 2), s.y(Math.min(b.avg, s.yMax))]);
    else flush();
  }
  flush();
  return { line, area };
}

function yGuides(s) {
  const marks = [svgEl("line", { class: "axis", x1: s.x0, x2: s.x1, y1: s.y0, y2: s.y0 })];
  marks.push(svgEl("text", { class: "ylabel", x: s.x0 - 6, y: s.y0 + 4, text: "0" }));
  for (const g of GUIDES) {
    const y = s.y(g.ms).toFixed(1);
    marks.push(svgEl("line", { class: `guide guide-${g.word}`, x1: s.x0, x2: s.x1, y1: y, y2: y }));
    marks.push(svgEl("text", { class: "ylabel", x: s.x0 - 6, y: +y + 4, text: `${g.ms} ms` }));
    marks.push(svgEl("text", { class: `glabel glabel-${g.word}`, x: s.x1 - 2, y: y - 4, text: g.word }));
  }
  return marks;
}

function xTicks(s) {
  const pxPerMinute = ((s.x1 - s.x0) * 60000) / (s.t1 - s.t0);
  const step = (TICK_MINUTES.find((m) => m * pxPerMinute >= 64) || 60) * 60000;
  const marks = [];
  for (let t = Math.ceil(s.t0 / step) * step; t <= s.t1; t += step) {
    const x = s.x(t).toFixed(1);
    marks.push(svgEl("line", { class: "tick", x1: x, x2: x, y1: s.y0, y2: s.y0 + 4 }));
    marks.push(svgEl("text", { class: "xlabel", x, y: s.y0 + 16, text: shortClock(t) }));
  }
  return marks;
}

function shortClock(t) {
  return new Date(t).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

function lossTicks(timeline, s) {
  const width = Math.max(2, ((s.x1 - s.x0) * s.span) / (s.t1 - s.t0) - 1);
  return timeline.filter((b) => b.lost > 0).map((b) => {
    const frac = b.sent ? b.lost / b.sent : 1;
    const h = 6 + frac * (s.y0 - s.y1 - 6);
    return svgEl("rect", { class: "loss", x: s.x(b.t).toFixed(1), y: (s.y0 - h).toFixed(1), width, height: h },
      svgEl("title", { text: `${b.lost} of ${b.sent} pings lost at ${formatClock(b.t)}` }));
  });
}

function band(s, from, to, cls) {
  const a = Math.max(s.x0, s.x(Math.min(from, to)));
  const b = Math.min(s.x1, s.x(Math.max(from, to)));
  const width = Math.max(0, b - a);
  return svgEl("rect", { class: cls, x: a.toFixed(1), y: s.y1, width: width.toFixed(1), height: s.y0 - s.y1 });
}

function selectionBand(chart, s) {
  const sel = chart.drag || chart.pending;
  return sel ? band(s, sel.from, sel.to, "selection") : svgEl("g");
}

function expirePending(chart) {
  const p = chart.pending;
  if (p && (chart.snap.window === "custom" || Date.now() - p.at > PENDING_MS)) chart.pending = null;
}

function attachDrag(chart) {
  const svg = chart.svg;
  svg.addEventListener("pointerdown", (e) => startDrag(chart, e));
  svg.addEventListener("pointermove", (e) => moveDrag(chart, e));
  svg.addEventListener("pointerup", (e) => endDrag(chart, e));
  svg.addEventListener("pointercancel", () => { chart.drag = null; draw(chart); });
}

function pointerTime(chart, e) {
  const s = chart.scale;
  const px = e.clientX - chart.svg.getBoundingClientRect().left;
  return s.t(Math.min(s.x1, Math.max(s.x0, px)));
}

function startDrag(chart, e) {
  if (!chart.interactive || !chart.scale || e.button !== 0) return;
  const t = pointerTime(chart, e);
  chart.drag = { from: t, to: t };
  chart.svg.setPointerCapture(e.pointerId);
  draw(chart);
}

function moveDrag(chart, e) {
  if (!chart.drag) return;
  chart.drag.to = pointerTime(chart, e);
  draw(chart);
}

function endDrag(chart, e) {
  if (!chart.drag) return;
  chart.drag.to = pointerTime(chart, e);
  const from = Math.round(Math.min(chart.drag.from, chart.drag.to));
  const to = Math.round(Math.max(chart.drag.from, chart.drag.to));
  chart.drag = null;
  const s = chart.scale;
  if (s.x(to) - s.x(from) >= MIN_DRAG_PX) {
    chart.pending = { from, to, at: Date.now() };
    chart.onSelect(from, to);
  }
  draw(chart);
}
