// Page entry: chooses live or report mode and wires the views together.

import { createChartControls } from "./chart-controls.js";
import { renderNotices, renderSites, renderStatus, renderVerdict } from "./header.js";
import { connectStream, fetchReport, postSite, postWindow } from "./live.js";
import { createNearMap } from "./map-near.js";
import { createUsMap } from "./map-us.js";
import { normalizeSnapshot } from "./model.js";
import { createMotion } from "./motion.js";
import { decodeReport, reportBanner, reportFromHash, reportStatus, summaryText } from "./report.js";
import { renderStretches } from "./stretches.js";
import { renderHops } from "./table.js";
import { createTimeline } from "./timeline.js";
import { $, formatClock, formatDuration, formatShortRange } from "./view.js";

const VIEWS = ["loading", "error", "empty", "content"];

const state = { payload: null, sites: [], site: 0, alive: null, firstSeen: 0, report: null };
const ui = { near: null, us: null, motion: null, timeline: null, controls: null };

boot();

function boot() {
  window.addEventListener("hashchange", () => location.reload());
  ui.near = createNearMap($("near-map"), $("near-title"));
  ui.us = createUsMap($("us-map"));
  ui.motion = createMotion([$("near-map"), $("us-map")]);
  const encoded = reportFromHash();
  if (encoded === null) bootLive();
  else bootReport(encoded);
}

// Report mode

async function bootReport(encoded) {
  document.body.dataset.mode = "report";
  ui.timeline = createTimeline($("timeline"), null);
  ui.controls = createChartControls(ui.timeline, null);
  renderStatus("report", "Report");
  try {
    state.report = await decodeReport(encoded);
  } catch (err) {
    $("error-detail").textContent = `Details: ${err.message}`;
    showView("error");
    return;
  }
  const snap = state.report;
  $("copy-summary").addEventListener("click", copySummary);
  renderSites($("sites"), [{ name: snap.site, grade: snap.verdict.grade }], 0, () => {});
  renderStatus("report", reportStatus(snap));
  renderNotices([{ kind: "report", text: reportBanner(snap) }, ...modeNotices(snap)]);
  showSnapshot(snap, snap.to, false);
}

function copySummary() {
  showLink(location.href);
  copyText(summaryText(state.report, location.href), "Summary copied");
}

// Live mode

function bootLive() {
  ui.timeline = createTimeline($("timeline"), (from, to) => chooseWindow("custom", { from, to }));
  ui.controls = createChartControls(ui.timeline, chooseWindow);
  $("bad-now").addEventListener("click", () => chooseWindow("bad"));
  $("copy-report").addEventListener("click", copyReport);
  $("site-form").addEventListener("submit", submitSite);
  connectStream(onPayload, onAlive);
  setInterval(updateStatus, 1000);
}

function onPayload(payload) {
  state.payload = payload;
  state.sites = (payload.sites || []).map(normalizeSnapshot);
  state.firstSeen ||= Date.now();
  state.site = Math.min(state.site, Math.max(0, state.sites.length - 1));
  renderLive();
}

function onAlive(alive) {
  state.alive = alive;
  updateStatus();
  renderLive();
}

function renderLive() {
  const snap = state.sites[state.site] || null;
  const tabs = state.sites.map((s) => ({ name: s.site, grade: s.verdict.grade }));
  renderSites($("sites"), tabs, state.site, pickSite);
  renderNotices(liveNotices(snap));
  if (!state.payload) return showView(state.alive === false ? "none" : "loading");
  if (!snap) return showView("empty");
  showSnapshot(snap, state.payload.now, state.alive !== false);
}

function liveNotices(snap) {
  const notices = [];
  if (state.alive === false) {
    notices.push({ kind: "stopped", text: "Stopped — the program is no longer running. The last results stay on screen." });
  }
  if (snap) notices.push(...modeNotices(snap));
  return notices;
}

function modeNotices(snap) {
  if (!snap.mode.partial) return [];
  const text = snap.mode.hint || "Some routers along the way can't be measured from this computer.";
  return [{ kind: "partial", text }];
}

function updateStatus() {
  if (state.alive === null) return renderStatus("connecting", "Connecting…");
  if (state.alive === false) return renderStatus("stopped", "Stopped");
  const started = state.payload?.started || state.firstSeen || Date.now();
  renderStatus("live", `Live ${formatDuration(Date.now() - started)}`);
}

function pickSite(index) {
  state.site = index;
  say("");
  $("link-box").hidden = true;
  renderLive();
}

async function chooseWindow(kind, range) {
  try {
    await postWindow(state.site, kind, range);
    say(kind === "bad" ? `Marked ${formatClock(Date.now())}` : "");
  } catch (err) {
    say(`Couldn't reach the program: ${err.message}`);
  }
}

async function copyReport() {
  const button = $("copy-report");
  button.disabled = true;
  try {
    const report = await fetchReport(state.site);
    $("report-link").value = report.link;
    await copyText(`${report.link}\n\n${report.text}`, "Link copied");
  } catch (err) {
    say(`Couldn't get the report: ${err.message}`);
  } finally {
    button.disabled = state.alive === false;
  }
}

async function submitSite(event) {
  event.preventDefault();
  const host = $("site-host").value.trim();
  const name = $("site-name").value.trim() || host;
  const button = event.submitter || $("site-form").querySelector("button");
  button.disabled = true;
  try {
    await postSite(name, host);
    $("site-status").textContent = "Starting. The first results appear in a few seconds.";
  } catch (err) {
    $("site-status").textContent = `That didn't work: ${err.message}`;
    button.disabled = false;
  }
}

// Shared rendering

function showSnapshot(snap, now, live) {
  showView("content");
  renderVerdict(snap, live);
  renderStretches($("stretches"), snap);
  renderMaps(snap, live);
  const marks = ui.motion.marks(state.site, snap, live && document.body.dataset.mode === "live");
  renderHops($("hops"), snap, marks);
  ui.motion.ripple(marks.lost);
  $("window-chip-text").textContent = formatShortRange(snap.from, snap.to);
  ui.timeline.render(snap, now, live);
  ui.controls.render(snap);
}

function renderMaps(snap, live) {
  const motion = live && document.body.dataset.mode === "live";
  const near = ui.near.render(snap, { live: motion });
  const us = ui.us.render(snap, { live: motion, near });
  $("near-figure").hidden = !near;
  $("maps").hidden = !near && !us;
  $("map-key").hidden = !near && !us;
}

function showView(name) {
  for (const view of VIEWS) $(view).hidden = view !== name;
}

function say(text) {
  $("action-status").textContent = text;
}

function showLink(link) {
  $("report-link").value = link;
  $("link-box").hidden = false;
}

async function copyText(text, okMessage) {
  try {
    await navigator.clipboard.writeText(text);
    say(okMessage);
  } catch {
    $("link-box").hidden = false;
    say("Your browser didn't allow copying. Select the link below and copy it instead.");
    $("report-link").select();
  }
}
