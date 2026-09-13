// The auth/session layer now persists through @cairn/shared's injected
// StorageAdapter (task_47c1), which the app wires up in src/config/init.ts at
// startup. Tests never import that entry point, so wire the same AsyncStorage
// adapter (itself mocked via jest.config moduleNameMapper) here.
import './src/config/init';
