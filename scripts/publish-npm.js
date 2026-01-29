#!/usr/bin/env node

/**
 * Script to publish n9e-mcp-server to npm
 *
 * Usage:
 *   node scripts/publish-npm.js <version>
 *
 * Example:
 *   node scripts/publish-npm.js 0.1.0
 *
 * This script will:
 * 1. Update version in all package.json files
 * 2. Download binaries from GitHub Release
 * 3. Publish all packages to npm
 */

const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

const NPM_DIR = path.join(__dirname, "..", "npm");
const GITHUB_REPO = "n9e/n9e-mcp-server";

// Mapping from npm platform names to goreleaser archive names
const PLATFORM_MAP = {
  "darwin-arm64": { os: "darwin", arch: "arm64", ext: "" },
  "darwin-x64": { os: "darwin", arch: "amd64", ext: "" },
  "linux-arm64": { os: "linux", arch: "arm64", ext: "" },
  "linux-x64": { os: "linux", arch: "amd64", ext: "" },
  "win32-arm64": { os: "windows", arch: "arm64", ext: ".exe" },
  "win32-x64": { os: "windows", arch: "amd64", ext: ".exe" },
};

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function downloadReleaseAsset(version, assetName, destDir, retries = 10, retryDelay = 30000) {
  for (let i = 0; i < retries; i++) {
    try {
      execSync(
        `gh release download v${version} --repo ${GITHUB_REPO} --pattern "${assetName}" --dir "${destDir}" --clobber`,
        { stdio: "inherit" }
      );
      return;
    } catch (e) {
      if (i < retries - 1) {
        console.log(`    Download failed, retrying in ${retryDelay / 1000}s... (${retries - i - 1} retries left)`);
        await sleep(retryDelay);
      } else {
        throw new Error(`Failed to download ${assetName} after ${retries} attempts`);
      }
    }
  }
}

async function extractTarGz(archivePath, destDir, binaryName) {
  execSync(`tar -xzf "${archivePath}" -C "${destDir}" ${binaryName}`, { stdio: "inherit" });
}

async function extractZip(archivePath, destDir, binaryName) {
  execSync(`unzip -o "${archivePath}" ${binaryName} -d "${destDir}"`, { stdio: "inherit" });
}

function updatePackageVersion(packageDir, version) {
  const packageJsonPath = path.join(packageDir, "package.json");
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, "utf8"));
  packageJson.version = version;

  // Update optionalDependencies versions if present
  if (packageJson.optionalDependencies) {
    for (const dep of Object.keys(packageJson.optionalDependencies)) {
      packageJson.optionalDependencies[dep] = version;
    }
  }

  fs.writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, 2) + "\n");
}

async function main() {
  const version = process.argv[2];
  if (!version) {
    console.error("Usage: node publish-npm.js <version>");
    console.error("Example: node publish-npm.js 0.1.0");
    process.exit(1);
  }

  const dryRun = process.argv.includes("--dry-run");
  const skipDownload = process.argv.includes("--skip-download");

  console.log(`Publishing version ${version}${dryRun ? " (dry run)" : ""}`);

  // Get all package directories
  const packages = fs.readdirSync(NPM_DIR).filter((name) => {
    const stat = fs.statSync(path.join(NPM_DIR, name));
    return stat.isDirectory() && fs.existsSync(path.join(NPM_DIR, name, "package.json"));
  });

  // Update versions in all packages
  console.log("\nUpdating package versions...");
  for (const pkg of packages) {
    updatePackageVersion(path.join(NPM_DIR, pkg), version);
    console.log(`  Updated ${pkg}`);
  }

  // Download and extract binaries for platform packages
  if (!skipDownload) {
    console.log("\nDownloading binaries from GitHub Release...");
    const tempDir = path.join(NPM_DIR, ".tmp");
    if (!fs.existsSync(tempDir)) {
      fs.mkdirSync(tempDir);
    }

    for (const [platform, info] of Object.entries(PLATFORM_MAP)) {
      const packageDir = path.join(NPM_DIR, `n9e-mcp-server-${platform}`);
      if (!fs.existsSync(packageDir)) continue;

      const archiveExt = info.os === "windows" ? "zip" : "tar.gz";
      const archiveName = `n9e-mcp-server-v${version}-${info.os}-${info.arch}.${archiveExt}`;
      const archivePath = path.join(tempDir, archiveName);

      console.log(`  Downloading ${archiveName}...`);
      try {
        await downloadReleaseAsset(version, archiveName, tempDir);
      } catch (e) {
        console.error(`  Failed to download ${archiveName}: ${e.message}`);
        process.exit(1);
      }

      const binaryName = `n9e-mcp-server${info.ext}`;
      console.log(`  Extracting ${binaryName}...`);
      if (archiveExt === "zip") {
        await extractZip(archivePath, packageDir, binaryName);
      } else {
        await extractTarGz(archivePath, packageDir, binaryName);
      }

      // Make binary executable on Unix
      if (info.ext === "") {
        fs.chmodSync(path.join(packageDir, binaryName), 0o755);
      }
    }

    // Cleanup temp directory
    fs.rmSync(tempDir, { recursive: true });
  }

  // Publish packages (platform packages first, then main package)
  console.log("\nPublishing packages to npm...");
  const platformPackages = packages.filter((p) => p !== "n9e-mcp-server");
  const publishOrder = [...platformPackages, "n9e-mcp-server"];

  for (const pkg of publishOrder) {
    const packageDir = path.join(NPM_DIR, pkg);
    console.log(`  Publishing ${pkg}...`);
    try {
      const cmd = dryRun ? "npm publish --access public --dry-run" : "npm publish --access public";
      execSync(cmd, { cwd: packageDir, stdio: "inherit" });
    } catch (e) {
      console.error(`  Failed to publish ${pkg}`);
      process.exit(1);
    }
  }

  console.log("\nDone!");
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
