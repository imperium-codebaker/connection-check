// Over time: the site's delay, loss or jitter in 10-second slices, with the blue window,
// lost pings, route changes, a hover readout and drag-to-select.

import {
  changeMarks, guides, lostMarks, readout, selection, series, timeTicks, valueMax, windowBand,
} from "./timeline-marks.js";
import { reducedMotion, svgEl } from "./view.js";

const MARGIN = { left: 52, right: 12, top: 18, bottom: 24 };
const MIN_DRAG_PX = 6;
const PENDING_MS = 4000;
const SLOT_MS = 10000;

/**
 * Create the chart on an SVG element. onSelect(from, to) gets drag selections in Unix ms;
 * without it the chart is read-only.
 */
export function createTimeline(svg, onSelect) {
  const chart = {
    svg, onSelect, snap: null, now: 0, live: false, measure: "avg", minutes: 15,
    drag: null, pending: null, hover: null, scale: null,
  };
  chart.plot = svgEl("g");
  chart.pulse = svgEl("circle", { class: "pulse-ring newest", r: 5, "aria-hidden": "true" });
  svg.replaceChildren(chart.plot, chart.pulse);
  attachPointer(chart);
  new ResizeObserver(() => draw(chart)).observe(svg);
  return {
    render(snap, now, live) {
      Object.assign(chart, { snap, now, live });
      expirePending(chart);
      draw(chart);
    },
    /** Show "avg", "loss" or "jitter". */
    setMeasure(measure) {
      chart.measure = measure;
      draw(chart);
    },
    /** Show the last 15 or 60 minutes. */
    setMinutes(minutes) {
      chart.minutes = minutes;
      draw(chart);
    },
    /** Return the shown time range as {from, to}. */
    shown() {
      return chart.scale ? { from: Math.round(chart.scale.t0), to: Math.round(chart.scale.t1) } : null;
    },
  };
}

function interactive(chart) {
  return chart.live && Boolean(chart.onSelect);
}

function draw(chart) {
  if (!chart.snap || !chart.svg.clientWidth) return;
  const s = (chart.scale = makeScale(chart));
  const shown = chart.snap.timeline.filter((b) => b.t + s.span > s.t0 && b.t < s.t1);
  chart.svg.setAttribute("viewBox", `0 0 ${s.width} ${s.height}`);
  chart.svg.classList.toggle("selectable", interactive(chart));
  const sel = chart.drag || chart.pending;
  chart.plot.replaceChildren(
    ...windowBand(s, chart.snap.from, chart.snap.to),
    ...guides(s), ...timeTicks(s),
    ...series(shown, s),
    ...lostMarks(shown, s),
    ...changeMarks(chart.snap.routeChanges, s),
    sel ? selection(s, sel.from, sel.to) : svgEl("g"),
    ...(chart.hover !== null && !chart.drag ? readout(shown, s, chart.hover) : []));
  placePulse(chart, shown, s);
}

function makeScale(chart) {
  const width = chart.svg.clientWidth;
  const height = chart.svg.clientHeight || 220;
  const { timeline } = chart.snap;
  const span = timeline.length > 1 ? timeline[1].t - timeline[0].t : SLOT_MS;
  const end = timeline.length ? timeline.at(-1).t + span : chart.now;
  const t1 = Math.max(end, chart.live ? chart.now : 0);
  const t0 = t1 - chart.minutes * 60000;
  const yMax = valueMax(timeline.filter((b) => b.t >= t0), chart.measure);
  const x0 = MARGIN.left, x1 = width - MARGIN.right;
  const y0 = height - MARGIN.bottom, y1 = MARGIN.top;
  return {
    width, height, t0, t1, span, yMax, x0, x1, y0, y1, measure: chart.measure,
    slot: Math.max(1.5, ((x1 - x0) * span) / (t1 - t0) - 0.5),
    x: (t) => x0 + ((t - t0) / (t1 - t0)) * (x1 - x0),
    y: (v) => y0 - (v / yMax) * (y0 - y1),
    t: (px) => t0 + ((px - x0) / (x1 - x0)) * (t1 - t0),
  };
}

/** Keep the pulse on the newest point while live; the element persists so its animation runs on. */
function placePulse(chart, shown, s) {
  const newest = shown.findLast((b) => b.sent > b.lost && b.avg > 0);
  const on = chart.live && chart.measure === "avg" && newest && !reducedMotion();
  chart.pulse.toggleAttribute("hidden", !on);
  if (!on) return;
  chart.pulse.setAttribute("cx", s.x(newest.t + s.span / 2).toFixed(1));
  chart.pulse.setAttribute("cy", s.y(Math.min(newest.avg, s.yMax)).toFixed(1));
}

function expirePending(chart) {
  const p = chart.pending;
  if (p && (chart.snap.window === "custom" || Date.now() - p.at > PENDING_MS)) chart.pending = null;
}

function attachPointer(chart) {
  const svg = chart.svg;
  svg.addEventListener("pointerdown", (e) => startDrag(chart, e));
  svg.addEventListener("pointermove", (e) => movePointer(chart, e));
  svg.addEventListener("pointerup", (e) => endDrag(chart, e));
  svg.addEventListener("pointerleave", () => { chart.hover = null; draw(chart); });
  svg.addEventListener("pointercancel", () => { chart.drag = null; draw(chart); });
}

function pointerTime(chart, e) {
  const s = chart.scale;
  const px = e.clientX - chart.svg.getBoundingClientRect().left;
  return s.t(Math.min(s.x1, Math.max(s.x0, px)));
}

function startDrag(chart, e) {
  if (!interactive(chart) || !chart.scale || e.button !== 0) return;
  const t = pointerTime(chart, e);
  chart.drag = { from: t, to: t };
  chart.svg.setPointerCapture(e.pointerId);
  draw(chart);
}

function movePointer(chart, e) {
  if (!chart.scale) return;
  if (chart.drag) chart.drag.to = pointerTime(chart, e);
  else chart.hover = pointerTime(chart, e);
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
