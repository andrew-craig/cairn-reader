import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { configureStorage, configureDefaultServerUrl } from '@cairn/shared';
import { localStorageAdapter } from './services/storage';
import App from './App';
import './index.css';

// Wire the shared config/service layer to the browser's localStorage backend
// and inject the web-specific default URL resolver (Vite env var or serving origin).
configureStorage(localStorageAdapter);
configureDefaultServerUrl(() => {
  const envUrl = import.meta.env.VITE_API_URL?.trim().replace(/\/+$/, '');
  if (envUrl) return envUrl;
  return typeof window !== 'undefined' ? window.location.origin : '';
});

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
