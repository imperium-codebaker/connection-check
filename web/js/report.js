// Report links: decoding a snapshot from the URL hash and writing a plain-text summary.

import { KIND_LABELS, blamedRow, normalizeSnapshot, osLabel, routerName } from "./model.js";
import {
  formatClock, formatDateTime, formatLoss, formatMs, formatShortRange, gradeWord, windowLabel,
} from "./view.js";

const PREFIX = "#r=";

/** Return the encoded report in the URL hash, or null when the page is in live mode. */
export function reportFromHash(hash = location.hash) {
  return hash.startsWith(PREFIX) ? hash.slice(PREFIX.length) : null;
}

/** Decode base64url(gzip(JSON snapshot)) into a normalized snapshot. */
export async function decodeReport(encoded) {
  if (typeof DecompressionStream === "undefined") {
    throw new Error("this browser cannot unpack report links; try a current Chrome, Edge, Firefox or Safari");
  }
  const bytes = base64urlBytes(encoded);
  const stream = new Blob([bytes]).stream().pipeThrough(new DecompressionStream("gzip"));
  let text;
  try {
    text = await new Response(stream).text();
  } catch {
    throw new Error("the link is incomplete (the packed data does not unpack)");
  }
  return normalizeSnapshot(JSON.parse(text));
}

function base64urlBytes(encoded) {
  let text = encoded;
  try {
    text = decodeURIComponent(encoded);
  } catch {
    // Keep the raw text; atob reports anything unusable.
  }
  const b64 = text.replace(/\s/g, "").replace(/-/g, "+").replace(/_/g, "/");
  if (!b64) throw new Error("the link has no report data after #r=");
  let binary;
  try {
    binary = atob(b64 + "=".repeat((4 - (b64.length % 4)) % 4));
  } catch {
    throw new Error("the link contains characters that are not part of a report");
  }
  return Uint8Array.from(binary, (c) => c.charCodeAt(0));
}

/** Return the banner line shown above a report. */
export function reportBanner(snap) {
  return `Report from ${formatDateTime(snap.to)}, ${osLabel(snap.os)}`;
}

/** Return the status pill text for a report: "Report · {window name} · {from}–{to}". */
export function reportStatus(snap) {
  return `Report · ${windowLabel(snap.window)} · ${formatShortRange(snap.from, snap.to)}`;
}

/** Return a plain-text summary of a report, ending with its link. */
export function summaryText(snap, href) {
  const d = snap.dest;
  const lines = [
    `Connection Check report: ${snap.site} (${snap.host})`,
    `${windowLabel(snap.window)}, ${formatDateTime(snap.from)} to ${formatClock(snap.to)}, ${osLabel(snap.os)}`,
    "",
    `${gradeWord(snap.verdict.grade).toUpperCase()}: ${snap.verdict.headline}`,
    snap.verdict.action,
    "",
    `Destination: ${formatMs(d.avg)} average, ${formatLoss(d.loss)} loss, ${formatMs(d.jitter)} jitter`,
    blameLine(snap),
    "",
    href,
  ];
  return lines.filter((line) => line !== null).join("\n");
}

function blameLine(snap) {
  const row = blamedRow(snap.verdict);
  if (row === null) return null;
  const hop = row === "dest" ? snap.dest : snap.hops.find((h) => h.ttl === row);
  if (!hop) return null;
  const where = row === "dest" ? "the destination" : `hop ${hop.ttl}`;
  const kind = KIND_LABELS[row === "dest" ? "site" : hop.owner];
  return `Trouble starts at ${where}: ${routerName(hop)}${kind ? ` (${kind})` : ""}`;
}
