import path from "node:path";
import { fileURLToPath } from "node:url";
import express from "express";
import { createProxyMiddleware } from "http-proxy-middleware";

const port = Number(process.env.PORT ?? 3000);
const apiTarget = process.env.API_TARGET ?? "http://localhost:8080";
const distDir = path.join(path.dirname(fileURLToPath(import.meta.url)), "dist");

const app = express();

// Reverse-proxy the Go API. Mounted at the root with pathFilter so the full
// request path is forwarded unchanged (http-proxy-middleware strips the
// express mount prefix otherwise). Responses stream through, so Server-Sent
// Events pass unbuffered.
app.use(
  createProxyMiddleware({
    target: apiTarget,
    changeOrigin: true,
    pathFilter: ["/api", "/healthz", "/swagger"],
  }),
);

app.use(express.static(distDir));

// SPA fallback: any other GET serves index.html so client-side routing works.
app.use((req, res, next) => {
  if (req.method !== "GET") {
    next();
    return;
  }
  res.sendFile(path.join(distDir, "index.html"));
});

app.listen(port, () => {
  console.log(`vcoach web listening on :${port} (api: ${apiTarget})`);
});
