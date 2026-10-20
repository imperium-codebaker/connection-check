// Buttons around the Over time chart: measure, time range, the blue-time choices and
// the Following live toggle.

import { $ } from "./view.js";

const FOLLOWING = new Set(["default", "minute", "all"]);

/**
 * Wire the chart's buttons. onWindow(kind, range) asks for a window; it is null in reports,
 * where only the measure and range buttons work.
 */
export function createChartControls(chart, onWindow) {
  const ctl = { chart, onWindow, snap: null };
  pressGroup($("measure-group"), "measure", (v) => chart.setMeasure(v));
  pressGroup($("range-group"), "range", (v) => {
    chart.setMinutes(Number(v));
    if (ctl.snap?.window === "all") askWhole(ctl);
  });
  if (onWindow) wireWindows(ctl);
  return {
    /** Show which window snap uses. */
    render(snap) {
      ctl.snap = snap;
      for (const b of $("window-group").querySelectorAll("button")) {
        b.setAttribute("aria-pressed", String(b.dataset.window === snap.window));
      }
      $("follow").setAttribute("aria-pressed", String(FOLLOWING.has(snap.window)));
    },
  };
}

function pressGroup(group, key, onPick) {
  group.addEventListener("click", (e) => {
    const button = e.target.closest("button");
    if (!button) return;
    for (const b of group.querySelectorAll("button")) b.setAttribute("aria-pressed", String(b === button));
    onPick(button.dataset[key]);
  });
}

function wireWindows(ctl) {
  $("window-group").addEventListener("click", (e) => {
    const kind = e.target.closest("button")?.dataset.window;
    if (kind === "all") askWhole(ctl);
    else if (kind) ctl.onWindow(kind);
  });
  $("follow").addEventListener("click", () => {
    const snap = ctl.snap;
    if (!snap) return;
    if (FOLLOWING.has(snap.window)) ctl.onWindow("custom", { from: snap.from, to: snap.to });
    else ctl.onWindow("default");
  });
}

function askWhole(ctl) {
  const shown = ctl.chart.shown();
  if (shown) ctl.onWindow("all", shown);
}
