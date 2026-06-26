// Metro configuration for the monorepo. apps/mobile is a workspace member
// (root package.json) and consumes @cairn/shared from source, so Metro must
// watch the repo root and resolve modules from both the app and root
// node_modules. See https://docs.expo.dev/guides/monorepos/.
const { getDefaultConfig } = require('expo/metro-config');
const path = require('path');

const projectRoot = __dirname;
const monorepoRoot = path.resolve(projectRoot, '../..');

const config = getDefaultConfig(projectRoot);

// 1. Watch all files within the monorepo (so changes to @cairn/shared are seen).
config.watchFolders = [monorepoRoot];

// 2. Resolve modules from the app's own node_modules first, then the hoisted
//    root node_modules.
config.resolver.nodeModulesPaths = [
  path.resolve(projectRoot, 'node_modules'),
  path.resolve(monorepoRoot, 'node_modules'),
];

module.exports = config;
