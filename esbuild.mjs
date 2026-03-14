import * as esbuild from "esbuild";
import fs from "fs";
import path from "path";

const outdir = "app/static";

const config = {
  entryPoints: [
    { in: "app/assets/js/app.js", out: "app" },
    { in: "app/assets/css/app.css", out: "app" },
  ],
  outdir,
  bundle: true,
  minify: true,
  sourcemap: true,
  legalComments: "linked",
  metafile: true,
};

// Generate a manifest mapping base names to hashed filenames.
// e.g. { "app.js": "app-HASH.js", "app.css": "app-HASH.css" }
function writeManifest(metafile) {
  const manifest = {};
  for (const [outPath, meta] of Object.entries(metafile.outputs)) {
    if (outPath.endsWith(".map") || outPath.endsWith(".LEGAL.txt")) continue;
    if (!meta.entryPoint) continue;
    const name = path.basename(outPath).replace(/-[A-Z0-9]{8}\./, ".");
    const rel = outPath.replace(`${outdir}/`, "");
    manifest[name] = rel;
  }
  fs.writeFileSync(path.join(outdir, "manifest.json"), JSON.stringify(manifest, null, 2));
}

if (process.argv.includes("--watch")) {
  // Watch mode: stable names so already-open tabs still work after a rebuild.
  // The Go server sets Cache-Control: no-cache in dev, so a manual reload
  // always fetches the latest file.
  const ctx = await esbuild.context({
    ...config,
    minify: false,
    sourcemap: true,
    entryNames: "[name]",
    logLevel: "info",
  });
  await ctx.watch();
} else {
  // Production build: content hashing for cache busting.
  // Clean output dir first.
  if (fs.existsSync(outdir)) {
    for (const f of fs.readdirSync(outdir)) {
      if (f === ".gitkeep") continue;
      fs.rmSync(path.join(outdir, f), { recursive: true });
    }
  }

  const result = await esbuild.build({
    ...config,
    entryNames: "[name]-[hash]",
  });
  writeManifest(result.metafile);
}
