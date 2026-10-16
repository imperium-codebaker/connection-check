// Page entry: chooses live or report mode and wires the views together.

import { renderNotices, renderSites, renderStatus, renderVerdict } from "./header.js";
import { connectStream, fetchReport, postSite, postWindow } from "./live.js";
import { createMap } from "./map.js";
import { normalizeSnapshot } from "./model.js";
import { decodeReport, reportBanner, reportFromHash, summaryText } from "./report.js";
import { renderHops } from "./table.js";
import { createTimeline } from "./timeline.js";
import { $, formatDuration } from "./view.js";

const VIEWS = ["loading", "error", "empty", "content"];

const WINDOW_MESSAGES = {
  bad: "Marked. The page now shows the minute around this moment. Copy the report for IT while it's fresh.",
  default: "Showing the last 5 minutes again.",
  custom: "Showing the range you selected.",
};

const state = { payload: null, sites: [], site: 0, alive: null, firstSeen: 0, report: null };
const ui = { map: null, timeline: null };

boot();

function boot() {
  window.addEventListener("hashchange", () => location.reload());
  ui.map = createMap($("map"), $("strip"));
  const encoded = reportFromHash();
  if (encoded === null) bootLive();
  else bootReport(encoded);
}

// Report mode

async function bootReport(encoded) {
  document.body.dataset.mode = "report";
  ui.timeline = createTimeline($("timeline"), null);
  renderStatus("report", "Report");
  try {
    state.report = await decodeReport(encoded);
  } catch (err) {
    $("error-detail").textContent = `Details: ${err.message}`;
    showView("error");
    return;
  }
  $("copy-summary").addEventListener("click", copySummary);
  renderSites($("sites"), [state.report.site], 0, () => {});
  showSnapshot(state.report, state.report.to, false);
}

function copySummary() {
  showLink(location.href);
  copyText(summaryText(state.report, location.href), "Copied the summary. Paste it wherever you need it.");
}

// Live mode

function bootLive() {
  ui.timeline = createTimeline($("timeline"), (from, to) => chooseWindow("custom", { from, to }));
  $("bad-now").addEventListener("click", () => chooseWindow("bad"));
  $("back-default").addEventListener("click", () => chooseWindow("default"));
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
  renderSites($("sites"), state.sites.map((s) => s.site), state.site, pickSite);
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
    say(WINDOW_MESSAGES[kind]);
  } catch (err) {
    say(`Couldn't reach the program: ${err.message}`);
  }
}

async function copyReport() {
  const button = $("copy-report");
  button.disabled = true;
  try {
    const report = await fetchReport(state.site);
    showLink(report.link);
    await copyText(`${report.link}\n\n${report.text}`, "Copied the report. Paste it into an email or chat to IT.");
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

function showSnapshot(snap, now, running) {
  const notices = document.body.dataset.mode === "report"
    ? [{ kind: "report", text: reportBanner(snap) }, ...modeNotices(snap)]
    : null;
  if (notices) renderNotices(notices);
  showView("content");
  renderVerdict(snap, running);
  renderHops($("hop-rows"), snap);
  ui.timeline.render(snap, now, running);
  ui.map.render(snap, running);
  $("bad-now").disabled = !running;
  $("copy-report").disabled = !running;
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
    say("Your browser didn't allow copying. Select the link below and copy it instead.");
    $("report-link").select();
  }
}
