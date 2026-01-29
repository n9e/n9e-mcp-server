const os = require("os");
const path = require("path");

const PLATFORMS = {
  darwin: {
    arm64: "@n9e/n9e-mcp-server-darwin-arm64",
    x64: "@n9e/n9e-mcp-server-darwin-x64",
  },
  linux: {
    arm64: "@n9e/n9e-mcp-server-linux-arm64",
    x64: "@n9e/n9e-mcp-server-linux-x64",
  },
  win32: {
    arm64: "@n9e/n9e-mcp-server-win32-arm64",
    x64: "@n9e/n9e-mcp-server-win32-x64",
  },
};

function getBinaryPath() {
  const platform = os.platform();
  const arch = os.arch();

  const platformPackages = PLATFORMS[platform];
  if (!platformPackages) {
    throw new Error(`Unsupported platform: ${platform}`);
  }

  const packageName = platformPackages[arch];
  if (!packageName) {
    throw new Error(`Unsupported architecture: ${arch} on ${platform}`);
  }

  try {
    const packagePath = require.resolve(packageName);
    const packageDir = path.dirname(packagePath);
    const binaryName = platform === "win32" ? "n9e-mcp-server.exe" : "n9e-mcp-server";
    return path.join(packageDir, binaryName);
  } catch (e) {
    throw new Error(
      `Could not find binary for ${platform}-${arch}. ` +
      `Please ensure @n9e/n9e-mcp-server is installed correctly.\n` +
      `Original error: ${e.message}`
    );
  }
}

module.exports = { getBinaryPath };
