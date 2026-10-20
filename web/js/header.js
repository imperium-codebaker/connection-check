// Top of the page: site tabs, live status, notices, and the verdict card.

import { $, el, gradeWord } from "./view.js";

/**
 * Render site tabs (two or more sites) or the single site name.
 * sites is [{name, grade}]; onPick(index) switches.
 */
export function renderSites(nav, sites, current, onPick) {
  const key = JSON.stringify([sites, current]);
  if (nav.dataset.key === key) return;
  nav.dataset.key = key;
  if (sites.length < 2) {
    nav.replaceChildren(...sites.map((s) => el("span", { class: "site-name", text: s.name })));
    return;
  }
  nav.replaceChildren(...sites.map((s, i) => siteTab(s, i === current, () => onPick(i))));
}

function siteTab(site, active, onClick) {
  const grade = site.grade || "measuring";
  const button = el("button", { type: "button", class: "tab", "aria-pressed": String(active) },
    el("span", { class: `dot g-${grade}`, "aria-hidden": "true" }),
    site.name,
    el("span", { class: "visually-hidden", text: `, ${gradeWord(grade)}` }));
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

/** Render the verdict card; running enables the live buttons. */
export function renderVerdict(snap, running) {
  const grade = snap.verdict.grade || "measuring";
  const badge = $("grade");
  badge.className = `grade-badge g-${grade}`;
  badge.textContent = gradeWord(grade);
  $("verdict").dataset.grade = grade;
  $("headline").textContent = snap.verdict.headline || `Measuring your connection to ${snap.site}…`;
  $("action").textContent = snap.verdict.action;
  $("bad-now").disabled = !running || grade === "measuring";
  $("copy-report").disabled = !running;
}
