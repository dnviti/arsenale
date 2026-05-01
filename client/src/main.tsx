/* eslint-disable react-refresh/only-export-components */
import React, { useEffect } from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import { TooltipProvider } from '@/components/ui/tooltip';
import { useThemeStore } from './store/themeStore';
import { themes } from './theme/index';
import { applyDocumentTheme } from './theme/documentTheme';
import './fonts';
import './global.css';

function Root() {
  const themeName = useThemeStore((s) => s.themeName);
  const mode = useThemeStore((s) => s.mode);
  const theme = themes[themeName][mode];

  useEffect(() => {
    applyDocumentTheme(theme, themeName, mode);
  }, [mode, theme, themeName]);

  return (
    <TooltipProvider>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </TooltipProvider>
  );
}

// eslint-disable-next-line @typescript-eslint/no-non-null-assertion -- standard React entry point pattern
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <Root />
  </React.StrictMode>
);
