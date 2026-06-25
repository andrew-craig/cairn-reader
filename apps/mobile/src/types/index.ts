// Framework-agnostic data models come from the shared package (decision_9b2d ADR
// — single source of truth). navigation.ts (React Navigation param lists) is
// client-specific and stays local.
export * from '@cairn/shared';
export * from './navigation';
