import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App.js';
import './index.css';

console.log('Renderer process loaded!');
console.log('Checking window.electronAPI:', window.electronAPI);

if (window.electronAPI) {
  console.log('Platform:', window.electronAPI.platform);
  console.log('Electron version:', window.electronAPI.version);
  console.log('loadNotes method available:', typeof window.electronAPI.loadNotes);
} else {
  console.error('ERROR: window.electronAPI is not defined!');
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

