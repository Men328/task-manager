import { AppShell } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import { Outlet } from 'react-router-dom';

import { useProjectGate } from '../../context/project/useProjectGate';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../common/States';

import classes from './AppLayout.module.css';
import { Sidebar } from './Sidebar';
import { TopBar } from './TopBar';

export function AppLayout() {
  const [navOpened, { toggle, close }] = useDisclosure(false);
  const { t } = useTranslation();
  const gate = useProjectGate();

  return (
    <AppShell
      header={{ height: 'var(--tm-header-height)' }}
      navbar={{
        width: 'var(--tm-sidebar-width)',
        breakpoint: 'sm',
        collapsed: { mobile: !navOpened },
      }}
      padding={0}
      classNames={{ header: classes.header, navbar: classes.navbar, main: classes.main }}
    >
      <AppShell.Header>
        <TopBar navOpened={navOpened} onToggleNav={toggle} />
      </AppShell.Header>

      <AppShell.Navbar>
        <Sidebar onNavigate={close} />
      </AppShell.Navbar>

      <AppShell.Main>
        {gate.status === 'ready' ? <Outlet /> : null}
        {gate.status === 'loading' ? <LoadingBlock /> : null}
        {gate.status === 'empty' ? (
          <CenteredPanel title={t('project.emptyTitle')} description={t('project.emptyActive')} />
        ) : null}
        {gate.status === 'error' ? (
          <div style={{ padding: 24 }}>
            <ApiErrorAlert message={gate.message} onRetry={gate.retry} />
          </div>
        ) : null}
      </AppShell.Main>
    </AppShell>
  );
}

export default AppLayout;
