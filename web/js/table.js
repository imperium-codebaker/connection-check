// Hop table: one row per router, then the destination.

import { blamedRow, routerName } from "./model.js";
import { el, formatLoss, formatMs, gradeChip, ownerLabel } from "./view.js";

const STATS = [
  { key: "loss", text: (h) => formatLoss(h.loss), title: (h) => `${h.lost} of ${h.sent} pings lost` },
  { key: "latency", text: (h) => formatMs(h.avg), title: (h) => `best ${formatMs(h.best)}, worst ${formatMs(h.worst)}` },
  { key: "jitter", text: (h) => formatMs(h.jitter), title: () => "average change between replies" },
];

/** Render the rows of the hop table into tbody. */
export function renderHops(tbody, snap) {
  const blamed = blamedRow(snap.verdict);
  const rows = snap.hops.map((hop) => hopRow(hop, snap, blamed === hop.ttl, false));
  rows.push(hopRow(snap.dest, snap, blamed === "dest", true));
  tbody.replaceChildren(...rows);
}

function hopRow(hop, snap, blamed, isDest) {
  const classes = ["hop", isDest && "dest", !hop.answered && "silent", blamed && "blamed"];
  const measure = blamed ? snap.verdict.measure : "";
  return el("tr", { class: classes.filter(Boolean).join(" ") },
    el("td", { class: "num", text: isDest ? "Dest" : hop.ttl }),
    el("td", { class: "owner", text: ownerLabel(hop, snap.site) }),
    routerCell(hop, blamed, isDest),
    STATS.map((stat) => statCell(hop, stat, measure)),
    el("td", { class: "grade" }, hop.answered ? gradeChip(hop.grade) : "—"));
}

function routerCell(hop, blamed, isDest) {
  if (!hop.answered) {
    const text = isDest ? "no replies from the destination" : "doesn't answer pings — normal";
    return el("td", { class: "router" }, el("span", { class: "router-note", text }));
  }
  const tag = isDest ? "Problem is here" : "Problem starts here";
  return el("td", { class: "router" },
    el("span", { class: "router-name", text: routerName(hop) }),
    blamed && el("span", { class: "tag", text: tag }),
    el("span", { class: "router-detail", text: routerDetail(hop) }));
}

function routerDetail(hop) {
  const parts = [];
  if (hop.name && hop.addr) parts.push(hop.addr);
  if (hop.place) parts.push(hop.place.approx ? `near ${hop.place.city}` : hop.place.city);
  if (hop.asName && hop.owner !== "provider") parts.push(hop.asName);
  return parts.join(" · ");
}

function statCell(hop, stat, measure) {
  if (!hop.answered) return el("td", { class: "stat", text: "—" });
  const cls = stat.key === measure ? "stat measure" : "stat";
  return el("td", { class: cls, title: stat.title(hop), text: stat.text(hop) });
}
