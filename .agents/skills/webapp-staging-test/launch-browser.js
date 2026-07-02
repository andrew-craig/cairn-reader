// Reference launcher for driving the web app in this environment.
// Run with: NODE_PATH=/opt/node22/lib/node_modules node <your-script>.js
// Copy/adapt this pattern rather than requiring this file directly.
const { chromium } = require('playwright');

async function launch() {
  return chromium.launch({
    executablePath: '/opt/pw-browsers/chromium',
    args: [
      `--proxy-server=${process.env.HTTPS_PROXY}`,
      '--proxy-bypass-list=localhost;127.0.0.1',
      '--ssl-version-max=tls1.2',
    ],
  });
}

module.exports = { launch };
