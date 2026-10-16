// Usage: node make-us-outline.cjs <topojson-client.min.js> <states-albers-10m.json> <out.json>
const fs = require("fs");
const [clientPath, atlasPath, outPath] = process.argv.slice(2);
const topojson = require(clientPath);
const us = JSON.parse(fs.readFileSync(atlasPath, "utf8"));

const fmt = (p) => `${p[0].toFixed(1)},${p[1].toFixed(1)}`;

function lineString(points, close) {
  const out = [];
  let last = "";
  points.forEach((p, i) => {
    const s = fmt(p);
    if (s === last) return;
    out.push((i === 0 ? "M" : "L") + s);
    last = s;
  });
  return out.join("") + (close ? "Z" : "");
}

function polygons(geom) {
  const polys = geom.type === "Polygon" ? [geom.coordinates] : geom.coordinates;
  return polys.flatMap((rings) => rings.map((r) => lineString(r, true))).join("");
}

const nation = topojson.feature(us, us.objects.nation);
const nationPath = nation.features.map((f) => polygons(f.geometry)).join("");
const mesh = topojson.mesh(us, us.objects.states, (a, b) => a !== b);
const statesPath = mesh.coordinates.map((l) => lineString(l, false)).join("");

const out = {
  source: "us-atlas@3.0.1 states-albers-10m.json via topojson-client@3.1.0",
  width: 975,
  height: 610,
  nation: nationPath,
  states: statesPath,
};
fs.writeFileSync(outPath, JSON.stringify(out));
console.log("nation", nationPath.length, "states", statesPath.length);
