import React from 'react';
import ReactDOM from 'react-dom/client';
import App, { AppProps, DEFAULT_SOURCE } from './App';
import { setDocument, resetDocument } from './documents/editor/EditorContext';

import { CssBaseline, ThemeProvider } from '@mui/material';
import theme from './theme';

function isRendered(containerId: string): boolean {
  const container = document.getElementById(containerId);
  if (!container) {
    console.error(`Container with id ${containerId} not found`);
    return false;
  }
  return container.hasChildNodes();
}

function render(containerId: string, props: AppProps, force: boolean = false) {
  if (!isRendered(containerId) || force) {
    const container = document.getElementById(containerId);
    if (!container) return;

    ReactDOM.createRoot(container).render(
      <React.StrictMode>
        <ThemeProvider theme={theme}>
          <CssBaseline />
          <App {...props} />
        </ThemeProvider>
      </React.StrictMode>
    );
  }
}

export { App, setDocument, resetDocument, render, isRendered, DEFAULT_SOURCE };
