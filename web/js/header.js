// Top of the page: site tabs, live status, notices, and the verdict card.

import { $, el, formatRange, gradeWord, windowLabel } from "./view.js";

/** Render site tabs (two or more sites) or the single site name; onPick(index) switches. */
export function renderSites(nav, names, current, onPick) {
  const key = `${names.join("\n")}|${current}`;
  if (nav.dataset.key === key) return;
  nav.dataset.key = key;
  if (names.length < 2) {
    nav.replaceChildren(...names.map((name) => el("span", { class: "site-name", text: name })));
    return;
  }
  nav.replaceChildren(...names.map((name, i) => siteTab(name, i === current, () => onPick(i))));
}

function siteTab(name, active, onClick) {
  const button = el("button", { type: "button", class: "tab", "aria-pressed": String(active), text: name });
  button.addEventListener("click", onClick);
  return button;
}

/** Show the status pill: kind is "connecting", "live", "stopped" or "report". */
export function renderStatus(kind, text) {
  const node = $("status");
  node.className = `status status-${kind}`;
  node.textContent = text;
}

/** Render notices; each is {kind, text}. Unchanged notices are left alone for screen readers. */
export function renderNotices(notices) {
  const box = $("notices");
  const key = JSON.stringify(notices);
  if (box.dataset.key === key) return;
  box.dataset.key = key;
  box.replaceChildren(...notices.map((n) => el("p", {
    class: `notice notice-${n.kind}`,
    role: n.kind === "stopped" ? "alert" : null,
    text: n.text,
  })));
}

/** Render the verdict card and window indicator; interactive shows the window controls. */
export function renderVerdict(snap, interactive) {
  const grade = snap.verdict.grade || "measuring";
  const badge = $("grade");
  badge.className = `grade-badge g-${grade}`;
  badge.textContent = gradeWord(grade);
  $("verdict").dataset.grade = grade;
  $("headline").textContent = snap.verdict.headline || `Measuring your connection to ${snap.site}…`;
  $("action").textContent = snap.verdict.action;
  $("window-label").textContent = windowLabel(snap.window);
  $("window-range").textContent = formatRange(snap.from, snap.to);
  $("back-default").hidden = !interactive || snap.window === "default";
}
