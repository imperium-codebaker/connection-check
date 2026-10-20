// The route in stretches: one box per stretch, home to site.

import { KIND_LABELS, stretchTitle, troubleTag } from "./model.js";
import { el, formatMs, gradeWord } from "./view.js";

/** Render the stretch boxes into list; hides it for reports without stretches. */
export function renderStretches(list, snap) {
  list.hidden = !snap.stretches;
  if (!snap.stretches) return list.replaceChildren();
  list.replaceChildren(...snap.stretches.map((s, i) => box(s, i + 1, snap)));
}

function box(stretch, step, snap) {
  const isLink = stretch.kind === "link";
  const grade = isLink ? "link" : stretch.grade || "measuring";
  const classes = ["stretch", `kind-${stretch.kind}`, `g-${grade}`, stretch.blamed && "blamed"];
  return el("li", { class: classes.filter(Boolean).join(" ") },
    el("div", { class: "stretch-step" },
      el("span", { class: isLink ? "dash" : "dot", "aria-hidden": "true" }),
      `${step} · ${KIND_LABELS[stretch.kind]}`),
    el("div", { class: "stretch-name", text: stretchTitle(stretch, snap.site) }),
    el("div", { class: "stretch-city", text: cityText(stretch) }),
    stretch.blamed && el("span", { class: "tag", text: troubleTag(snap.verdict) }),
    el("div", { class: "stretch-figures" },
      el("span", { class: "stretch-ms", text: isLink ? `+${Math.round(stretch.ms)} ms` : msText(stretch) }),
      el("span", { class: "stretch-detail", text: isLink ? "No routers show" : detailText(stretch) })));
}

function cityText(stretch) {
  if (!stretch.city) return "—";
  return stretch.approx ? `≈ ${stretch.city}` : stretch.city;
}

function msText(stretch) {
  return stretch.ms > 0 ? formatMs(stretch.ms) : "—";
}

function detailText(stretch) {
  const routers = stretch.routers === 1 ? "1 router" : `${stretch.routers} routers`;
  return stretch.grade ? `${routers} · ${gradeWord(stretch.grade).toLowerCase()}` : routers;
}
