/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_AUTH_MODE: 'token' | 'signature';
  readonly VITE_API_KEY: string;
  readonly VITE_API_SECRET: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
