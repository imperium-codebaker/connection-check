// Shared view helpers: element builders and display text.

const SVG_NS = "http://www.w3.org/2000/svg";

const GRADE_WORDS = { good: "Good", fair: "Fair", poor: "Poor", measuring: "Measuring" };

const WINDOW_LABELS = {
  default: "Last 5 minutes",
  bad: "Bad moment (60 s)",
  custom: "Selected range",
};

/** Return the element with the given id. */
export const $ = (id) => document.getElementById(id);

/** Create an HTML element; attrs.text sets textContent, other keys become attributes. */
export function el(tag, attrs = {}, ...children) {
  return build(document.createElement(tag), attrs, children);
}

/** Create an SVG element the same way as el(). */
export function svgEl(tag, attrs = {}, ...children) {
  return build(document.createElementNS(SVG_NS, tag), attrs, children);
}

function build(node, attrs, children) {
  for (const [key, value] of Object.entries(attrs)) {
    if (value === null || value === undefined || value === false) continue;
    if (key === "text") node.textContent = String(value);
    else node.setAttribute(key, value === true ? "" : String(value));
  }
  for (const child of children.flat()) {
    if (child === null || child === undefined || child === false) continue;
    node.append(child instanceof Node ? child : String(child));
  }
  return node;
}

/** Return the display word for a grade; unknown grades read as "Measuring". */
export function gradeWord(grade) {
  return GRADE_WORDS[grade] || GRADE_WORDS.measuring;
}

/** Return a grade chip: a coloured dot plus the grade word. */
export function gradeChip(grade) {
  const key = GRADE_WORDS[grade] ? grade : "measuring";
  return el("span", { class: `chip g-${key}` },
    el("span", { class: "dot", "aria-hidden": "true" }), gradeWord(key));
}

/** Return who a hop belongs to, in plain words. */
export function ownerLabel(hop, siteName) {
  switch (hop.owner) {
    case "home": return "Home";
    case "provider": return hop.asName ? `Your provider (${hop.asName})` : "Your provider";
    case "handoff": return "Handoff";
    case "site": return `${siteName}'s end`;
    default: return "—";
  }
}

/** Format milliseconds: one decimal below 10 ms, whole numbers above. */
export function formatMs(ms) {
  if (!Number.isFinite(ms)) return "—";
  return ms < 10 ? `${ms.toFixed(1)} ms` : `${Math.round(ms)} ms`;
}

/** Format a loss percentage. */
export function formatLoss(pct) {
  if (!Number.isFinite(pct) || pct === 0) return "0%";
  return pct < 10 ? `${pct.toFixed(1)}%` : `${Math.round(pct)}%`;
}

/** Format Unix ms as a local clock time with seconds. */
export function formatClock(ms) {
  return new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

/** Format Unix ms as a local date and time. */
export function formatDateTime(ms) {
  return new Date(ms).toLocaleString([], { dateStyle: "medium", timeStyle: "short" });
}

/** Format a from–to pair of Unix ms as local clock times. */
export function formatRange(from, to) {
  return `${formatClock(from)}–${formatClock(to)}`;
}

/** Format an elapsed time as "4m12s" or "1h05m". */
export function formatDuration(ms) {
  const s = Math.max(0, Math.floor(ms / 1000));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const pad = (n) => String(n).padStart(2, "0");
  if (h) return `${h}h${pad(m)}m`;
  return m ? `${m}m${pad(s % 60)}s` : `${s}s`;
}

/** Return the label for a window kind. */
export function windowLabel(kind) {
  return WINDOW_LABELS[kind] || WINDOW_LABELS.default;
}
