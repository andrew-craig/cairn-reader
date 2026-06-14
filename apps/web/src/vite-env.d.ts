/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Optional backend base URL. Overrides the serving origin as the default
   *  backend — primarily for local development (see src/config/api.ts). */
  readonly VITE_API_URL?: string;
}
