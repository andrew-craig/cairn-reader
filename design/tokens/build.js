// Generates each app's color file from colors.json (the single source of the
// palette). Output files are committed; CI fails if they drift from the source.
// A native Swift app would add a platform here (e.g. an asset-catalog colorset
// format) without touching colors.json.
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
  },
});

await sd.buildAllPlatforms();
