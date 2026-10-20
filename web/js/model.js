// Snapshot shape helpers shared by the live page and report links.

const GRADE_RANK = { good: 1, fair: 2, poor: 3 };

const OS_NAMES = { darwin: "macOS", macos: "macOS", windows: "Windows", linux: "Linux" };

/** Fixed labels for each kind of stretch. */
export const KIND_LABELS = { home: "Home", provider: "Provider", handoff: "Handoff", link: "Private link", site: "Site" };

const KIND_NAMES = { home: "Your network", provider: "Your provider", handoff: "Other carriers", link: "Unseen stretch" };

const MEASURE_WORDS = { loss: "loss", latency: "delay", jitter: "jitter" };

const EMPTY_HOP = {
  ttl: 0, addr: "", name: "", asn: 0, asName: "", owner: "unknown", answered: false,
  sent: 0, lost: 0, loss: 0, avg: 0, best: 0, worst: 0, jitter: 0, grade: "", ignores: false, place: null,
};

const EMPTY_STRETCH = {
  kind: "site", name: "", city: "", approx: false, firstTtl: 0, lastTtl: 0, routers: 0, ms: 0, grade: "", blamed: false,
  near: false,
};

/**
 * Fill missing or null fields so the views can rely on the snapshot shape.
 * Version 1 reports carry no stretches; stretches is then null.
 */
export function normalizeSnapshot(snap) {
  if (!snap || typeof snap !== "object" || !snap.verdict) {
    throw new Error("the data is not a connection report");
  }
  const stretches = Array.isArray(snap.stretches) && snap.stretches.length ? snap.stretches : null;
  return {
    ...snap,
    mode: snap.mode || { name: "", partial: false },
    verdict: { grade: "measuring", headline: "", action: "", measure: "", owner: "", blameTtl: 0, ...snap.verdict },
    hops: (snap.hops || []).map((hop) => ({ ...EMPTY_HOP, ...hop })),
    dest: { ...EMPTY_HOP, ...(snap.dest || {}) },
    timeline: (snap.timeline || []).map((b) => ({ sent: 0, lost: 0, avg: 0, jitter: 0, ...b })),
    stretches: stretches && stretches.map((s) => ({ ...EMPTY_STRETCH, ...s })),
    routeChanges: snap.routeChanges || [],
  };
}

/** Return the hops in a stretch, with the destination last in the site stretch. */
export function stretchHops(snap, stretch) {
  const hops = stretch.firstTtl > 0
    ? snap.hops.filter((h) => h.ttl >= stretch.firstTtl && h.ttl <= stretch.lastTtl)
    : [];
  return stretch.kind === "site" ? [...hops, snap.dest] : hops;
}

/** Return the title of a stretch: its network names, or a fixed name for its kind. */
export function stretchTitle(stretch, siteName) {
  return stretch.name || KIND_NAMES[stretch.kind] || siteName;
}

/** Return the tag for the stretch or row where the trouble starts. */
export function troubleTag(verdict) {
  const word = MEASURE_WORDS[verdict.measure];
  return word ? `Trouble starts here (${word})` : "Trouble starts here";
}

/** Return the TTL of the row to highlight, "dest" for the destination, or null. */
export function blamedRow(verdict) {
  if (verdict.blameTtl > 0) return verdict.blameTtl;
  return verdict.owner === "site" ? "dest" : null;
}

/** Return the worst grade among answered hops, or "measuring" when none is graded. */
export function worstGrade(hops) {
  let worst = "";
  for (const hop of hops) {
    if (hop.answered && (GRADE_RANK[hop.grade] || 0) > (GRADE_RANK[worst] || 0)) worst = hop.grade;
  }
  return worst || "measuring";
}

/** Return "silent" for a hop that never answers, "ignores" for one rationing its replies, else its grade. */
export function hopState(hop) {
  if (!hop.answered) return "silent";
  return hop.ignores ? "ignores" : hop.grade || "measuring";
}

/** Return the key that identifies a hop's row across updates. */
export function rowKey(hop) {
  return `${hop.ttl}|${hop.addr}`;
}

/** Return the name to show for a hop's router. */
export function routerName(hop) {
  return hop.name || hop.addr || "unnamed router";
}

/** Return a readable operating system name. */
export function osLabel(os) {
  return OS_NAMES[String(os).toLowerCase()] || os || "unknown system";
}

/** Return the city of the first stretch past home that ends near the user, or "" when none has a place. */
export function nearCity(snap) {
  return snap.stretches?.find((s) => s.near && s.kind !== "home" && s.kind !== "link" && s.city)?.city || "";
}

/** Return the city of the site, from the destination or the site stretch. */
export function siteCity(snap) {
  return snap.dest.place?.city || snap.stretches?.at(-1)?.city || "";
}
