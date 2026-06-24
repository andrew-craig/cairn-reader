import { registerRootComponent } from 'expo';

import App from './App';

// Explicit entry point for the monorepo. The default `expo/AppEntry.js` resolves
// the app via a path relative to the expo package, which breaks once expo is
// hoisted to the workspace-root node_modules. Registering here keeps the entry
// anchored to this app regardless of where dependencies hoist.
registerRootComponent(App);
