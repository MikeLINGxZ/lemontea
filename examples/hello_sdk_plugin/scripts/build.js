const fs = require('fs');
const path = require('path');

const distDir = path.join(__dirname, '..', 'dist');
const srcViewsDir = path.join(__dirname, '..', 'src', 'views');
const distViewsDir = path.join(distDir, 'views');
const distSDKDir = path.join(distDir, 'sdk');
const sdkDir = path.join(__dirname, '..', '..', '..', 'packages', 'plugin-sdk');

function copyDir(src, dest) {
  fs.mkdirSync(dest, { recursive: true });
  for (const entry of fs.readdirSync(src, { withFileTypes: true })) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    if (entry.isDirectory()) {
      copyDir(srcPath, destPath);
      continue;
    }
    fs.copyFileSync(srcPath, destPath);
  }
}

fs.mkdirSync(distDir, { recursive: true });
fs.mkdirSync(distSDKDir, { recursive: true });
fs.writeFileSync(path.join(distDir, 'main.js'), "require('../src/main');\n");
copyDir(srcViewsDir, distViewsDir);
fs.copyFileSync(path.join(sdkDir, 'runtime.js'), path.join(distSDKDir, 'runtime.js'));
fs.copyFileSync(path.join(sdkDir, 'browser.js'), path.join(distViewsDir, 'sdk.js'));
