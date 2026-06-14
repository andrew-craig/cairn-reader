// Public surface of the @cairn/shared package: framework-agnostic types reused
// by both apps/mobile and apps/web. Server-URL configuration is app-specific
// (web defaults to its serving origin, mobile to a fixed backend) and lives in
// each app rather than here.
export * from './types';
