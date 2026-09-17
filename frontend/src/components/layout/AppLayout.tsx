import { AppShell } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { Outlet } from 'react-router-dom';

import classes from './AppLayout.module.css';
import { Sidebar } from './Sidebar';
import { TopBar } from './TopBar';

export function AppLayout() {
  const [navOpened, { toggle, close }] = useDisclosure(false);

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
        <Outlet />
      </AppShell.Main>
    </AppShell>
  );
}

export default AppLayout;
