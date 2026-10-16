// Snapshot shape helpers shared by the live page and report links.

const GRADE_RANK = { good: 1, fair: 2, poor: 3 };

const OS_NAMES = { darwin: "macOS", macos: "macOS", windows: "Windows", linux: "Linux" };

const EMPTY_HOP = {
  ttl: 0, addr: "", name: "", asn: 0, asName: "", owner: "unknown", answered: false,
  sent: 0, lost: 0, loss: 0, avg: 0, best: 0, worst: 0, jitter: 0, grade: "", place: null,
};

/** Fill missing or null fields so the views can rely on the snapshot shape. */
export function normalizeSnapshot(snap) {
  if (!snap || typeof snap !== "object" || !snap.verdict) {
    throw new Error("the data is not a connection report");
  }
  return {
    ...snap,
    mode: snap.mode || { name: "", partial: false },
    verdict: { grade: "measuring", headline: "", action: "", blameTtl: 0, ...snap.verdict },
    hops: (snap.hops || []).map((hop) => ({ ...EMPTY_HOP, ...hop })),
    dest: { ...EMPTY_HOP, ...(snap.dest || {}) },
    timeline: snap.timeline || [],
  };
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

/** Return the name to show for a hop's router. */
export function routerName(hop) {
  return hop.name || hop.addr || "unnamed router";
}

/** Return a readable operating system name. */
export function osLabel(os) {
  return OS_NAMES[String(os).toLowerCase()] || os || "unknown system";
}
