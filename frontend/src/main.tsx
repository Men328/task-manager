import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { Provider } from 'react-redux';
import { BrowserRouter } from 'react-router-dom';

import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';
import './styles/tokens.css';
import './styles/global.css';
import './i18n';

import App from './App';
import { store } from './context';
import { DEFAULT_COLOR_SCHEME, colorSchemeManager, theme } from './theme';

const container = document.getElementById('root');

if (!container) {
  throw new Error('Root element "#root" was not found in index.html');
}

createRoot(container).render(
  <StrictMode>
    <Provider store={store}>
      <MantineProvider
        theme={theme}
        defaultColorScheme={DEFAULT_COLOR_SCHEME}
        colorSchemeManager={colorSchemeManager}
      >
        <Notifications position="top-right" limit={5} />
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </MantineProvider>
    </Provider>
  </StrictMode>,
);
