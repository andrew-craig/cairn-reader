/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Optional backend base URL. Overrides the serving origin as the default
   *  backend — primarily for local development (see src/config/api.ts). */
  readonly VITE_API_URL?: string;
}

/** The web app's own version, injected from package.json at build time
 *  (see vite.config.ts). */
declare const __APP_VERSION__: string;
