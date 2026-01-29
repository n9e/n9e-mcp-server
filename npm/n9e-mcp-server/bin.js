#!/usr/bin/env node

const { spawnSync } = require("child_process");
const { getBinaryPath } = require("./index.js");

const binaryPath = getBinaryPath();
const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
  env: process.env,
});

if (result.error) {
  console.error("Failed to execute n9e-mcp-server:", result.error.message);
  process.exit(1);
}

process.exit(result.status ?? 1);
