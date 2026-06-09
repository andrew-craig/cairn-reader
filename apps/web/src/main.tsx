import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { configureStorage } from '@cairn/shared';
import { localStorageAdapter } from './services/storage';
import App from './App';
import './index.css';

// Wire the shared config/service layer to the browser's localStorage backend.
configureStorage(localStorageAdapter);

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
