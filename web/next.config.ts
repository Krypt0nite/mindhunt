import type { NextConfig } from "next";

// Static export: the Go server serves ./out. In development, run the Go
// server on :8080 and set NEXT_PUBLIC_API_BASE=http://localhost:8080.
const config: NextConfig = {
  output: "export",
  trailingSlash: true,
  reactStrictMode: true,
  images: { unoptimized: true },
};

export default config;
