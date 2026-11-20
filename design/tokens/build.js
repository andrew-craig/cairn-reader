// Generates each app's color file from colors.json (the single source of the
// palette). Output files are committed; CI fails if they drift from the source.
import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import StyleDictionary from 'style-dictionary';

const HEADER = 'Generated from design/tokens/colors.json by `npm run build -w @cairn/tokens`. Do not edit.';
const MODES = ['light', 'dark'];

// { light: { primary: '#…', … }, dark: { … } }, failing if the modes' token names differ.
function byMode(dictionary) {
  const modes = Object.fromEntries(MODES.map((mode) => [mode, {}]));
  for (const token of dictionary.allTokens) {
    const [mode, name] = token.path;
    modes[mode][name] = token.$value;
  }
  const names = MODES.map((mode) => Object.keys(modes[mode]).sort().join());
  if (new Set(names).size !== 1) throw new Error('colors.json: light and dark must define the same tokens');
  return modes;
}

const kebab = (name) => name.replace(/[A-Z]/g, (c) => `-${c.toLowerCase()}`);
const cssVars = (colors, indent) =>
  Object.entries(colors)
    .map(([name, value]) => `${indent}--color-${kebab(name)}: ${value};`)
    .join('\n');

StyleDictionary.registerFormat({
  name: 'css/color-schemes',
  format: ({ dictionary }) => {
    const { light, dark } = byMode(dictionary);
    return `/* ${HEADER} */
:root {
${cssVars(light, '  ')}
}

@media (prefers-color-scheme: dark) {
  :root {
${cssVars(dark, '    ')}
  }
}
`;
  },
});

StyleDictionary.registerFormat({
  name: 'typescript/color-schemes',
  format: ({ dictionary }) => `// ${HEADER}
export const Colors = ${JSON.stringify(byMode(dictionary), null, 2).replace(/"(\w+)":/g, '$1:').replace(/"/g, "'")};
`,
});

// Xcode's asset-catalog JSON: two-space indent and " : " separators.
const xcodeJSON = (value) => `${JSON.stringify(value, null, 2).replace(/": /g, '" : ')}\n`;
const XCODE_INFO = { author: 'xcode', version: 1 };

// '#RRGGBB' or '#RRGGBBAA' to an sRGB colorset entry.
function colorsetColor(hex) {
  const [red, green, blue] = [1, 3, 5].map((i) => `0x${hex.slice(i, i + 2)}`);
  const alpha = hex.length === 9 ? parseInt(hex.slice(7, 9), 16) / 255 : 1;
  return { 'color-space': 'srgb', components: { alpha: alpha.toFixed(3), blue, green, red } };
}

// One <token>.colorset per token with a light (any) and dark appearance. The catalog is
// wholly generated, so it is recreated each build and removed tokens disappear.
StyleDictionary.registerAction({
  name: 'ios/colorsets',
  do: (dictionary, platform) => {
    const catalog = platform.buildPath;
    rmSync(catalog, { recursive: true, force: true });
    mkdirSync(catalog, { recursive: true });
    writeFileSync(`${catalog}/Contents.json`, xcodeJSON({ info: XCODE_INFO }));
    const { light, dark } = byMode(dictionary);
    for (const name of Object.keys(light)) {
      mkdirSync(`${catalog}/${name}.colorset`);
      writeFileSync(
        `${catalog}/${name}.colorset/Contents.json`,
        xcodeJSON({
          colors: [
            { color: colorsetColor(light[name]), idiom: 'universal' },
            {
              appearances: [{ appearance: 'luminosity', value: 'dark' }],
              color: colorsetColor(dark[name]),
              idiom: 'universal',
            },
          ],
          info: XCODE_INFO,
        }),
      );
    }
  },
  undo: () => {},
});

const sd = new StyleDictionary({
  source: [new URL('./colors.json', import.meta.url).pathname],
  log: { verbosity: 'silent' },
  platforms: {
    web: {
      buildPath: new URL('../../apps/web/src/theme/', import.meta.url).pathname,
      files: [{ destination: 'colors.css', format: 'css/color-schemes' }],
    },
    mobile: {
      buildPath: new URL('../../apps/mobile/src/constants/', import.meta.url).pathname,
      files: [{ destination: 'colors.ts', format: 'typescript/color-schemes' }],
    },
    ios: {
      buildPath: new URL('../../apps/ios/CairnReader/Resources/Colors.xcassets', import.meta.url).pathname,
      files: [],
      actions: ['ios/colorsets'],
    },
  },
});

await sd.buildAllPlatforms();
