// Ping results by router: one group per stretch, or one plain list for older reports.

import {
  KIND_LABELS, blamedRow, hopState, routerName, rowKey, stretchHops, stretchTitle, troubleTag,
} from "./model.js";
import { el, formatLoss, formatMs, gradeChip, quietChip } from "./view.js";

const STATS = [
  { key: "loss", text: (h) => formatLoss(h.loss), title: (h) => `${h.lost} of ${h.sent} pings lost` },
  { key: "latency", text: (h) => formatMs(h.avg), title: (h) => `${formatMs(h.best)}–${formatMs(h.worst)}` },
  { key: "jitter", text: (h) => formatMs(h.jitter), title: () => "Average change between replies" },
];

/**
 * Render the table body groups. marks.fresh and marks.lost are sets of row keys to fade in
 * and to ripple.
 */
export function renderHops(table, snap, marks) {
  const ctx = { snap, marks, blamed: blamedRow(snap.verdict) };
  const groups = snap.stretches
    ? snap.stretches.map((s) => stretchGroup(s, ctx))
    : [el("tbody", {}, snap.hops.map((h) => hopRow(h, ctx)), hopRow(snap.dest, ctx))];
  table.querySelectorAll("tbody").forEach((body) => body.remove());
  table.append(...groups);
}

function stretchGroup(stretch, ctx) {
  const rows = stretchHops(ctx.snap, stretch).map((h) => hopRow(h, ctx));
  if (stretch.kind === "link") rows.push(linkRow(stretch));
  return el("tbody", { class: `group kind-${stretch.kind}` }, groupHead(stretch, ctx), rows);
}

function groupHead(stretch, ctx) {
  return el("tr", { class: "group-head" },
    el("th", { scope: "rowgroup", colspan: 7 },
      el("span", { class: "group-name", text: stretchTitle(stretch, ctx.snap.site) }),
      el("span", { class: "group-kind", text: KIND_LABELS[stretch.kind] }),
      stretch.blamed && el("span", { class: "tag", text: troubleTag(ctx.snap.verdict) })));
}

function linkRow(stretch) {
  return el("tr", { class: "hop link-row" },
    el("td", { class: "num", text: "—" }),
    el("td", { colspan: 6, text: `No routers answer · +${Math.round(stretch.ms)} ms` }));
}

function hopRow(hop, ctx) {
  const isDest = hop === ctx.snap.dest;
  const blamed = ctx.blamed === (isDest ? "dest" : hop.ttl);
  const key = rowKey(hop);
  const state = hopState(hop);
  const classes = ["hop", isDest && "dest", `state-${state}`, blamed && "blamed",
    ctx.marks.fresh.has(key) && "fresh", ctx.marks.lost.has(key) && "ripple"];
  const measure = blamed ? ctx.snap.verdict.measure : "";
  return el("tr", { class: classes.filter(Boolean).join(" "), "data-key": key },
    el("td", { class: "num", text: isDest ? "Site" : hop.ttl }),
    routerCell(hop),
    el("td", { class: "where", text: whereText(hop) }),
    STATS.map((stat) => statCell(hop, stat, measure, state)),
    el("td", { class: "grade" }, gradeCell(hop, state)));
}

function routerCell(hop) {
  if (!hop.answered && !hop.addr) {
    return el("td", { class: "router" }, el("span", { class: "router-note", text: "Doesn't answer pings" }));
  }
  return el("td", { class: "router" },
    el("span", { class: "router-name", text: routerName(hop) }),
    hop.name && hop.addr && el("span", { class: "router-detail", text: hop.addr }));
}

function whereText(hop) {
  const p = hop.place;
  if (!p) return "—";
  if (p.given) return `${p.city} (site list)`;
  return p.approx ? `≈ ${p.city}` : p.city;
}

function statCell(hop, stat, measure, state) {
  if (!hop.answered) return el("td", { class: "stat", text: "—" });
  const classes = ["stat", stat.key === measure && "measure", state === "ignores" && stat.key === "loss" && "quiet"];
  return el("td", { class: classes.filter(Boolean).join(" "), title: stat.title(hop), text: stat.text(hop) });
}

function gradeCell(hop, state) {
  if (state === "silent") return quietChip("No reply");
  if (state === "ignores") return quietChip("Ignores pings");
  return gradeChip(hop.grade);
}
