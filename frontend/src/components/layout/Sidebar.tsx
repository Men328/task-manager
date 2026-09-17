import { Box, Group, Stack, Text, ThemeIcon, UnstyledButton } from '@mantine/core';
import {
  IconArrowsExchange,
  IconChecklist,
  IconSparkles,
} from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router-dom';

import { useSidebarCounts } from '../../context';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import classes from './Sidebar.module.css';

type IconComponent = typeof IconChecklist;

interface NavEntry {
  labelKey: string;
  icon: IconComponent;
  to: string;
}

/** Chỉ giữ những mục phục vụ personal hub: task + cấu hình trạng thái. */
const MAIN_MENU: NavEntry[] = [
  { labelKey: 'nav.myTask', icon: IconChecklist, to: '/' },
  { labelKey: 'nav.statusesLifecycle', icon: IconArrowsExchange, to: '/statuses' },
];

function SectionLabel({ children }: { children: string }) {
  return (
    <Text
      fz={10}
      fw={700}
      tt="uppercase"
      c={tokens.textFaint}
      className={classes.sectionLabel}
      px={10}
      pt={18}
      pb={6}
    >
      {children}
    </Text>
  );
}

function NavItem({
  entry,
  active,
  onSelect,
}: {
  entry: NavEntry;
  active: boolean;
  onSelect: (to: string) => void;
}) {
  const { t } = useTranslation();
  const { labelKey, icon: Icon, to } = entry;

  return (
    <UnstyledButton
      className={classes.navItem}
      data-active={active}
      onClick={() => onSelect(to)}
    >
      <Group gap={10} wrap="nowrap">
        <Icon size={17} stroke={1.8} style={{ color: active ? tokens.brand : tokens.textMuted }} />
        <Text fz={13} fw={active ? 700 : 500} c={active ? tokens.brand : tokens.navText}>
          {t(labelKey)}
        </Text>
      </Group>
    </UnstyledButton>
  );
}

function WorkspaceHeader() {
  const { t } = useTranslation();

  return (
    <Group gap={10} wrap="nowrap" px={4} pt={4}>
      <ThemeIcon
        size={30}
        radius={9}
        variant="gradient"
        gradient={{ from: tokens.brandFrom, to: tokens.brandTo, deg: 140 }}
      >
        <IconSparkles size={17} />
      </ThemeIcon>
      <Text fz={14} fw={700} c={tokens.text} className={classes.workspaceTitle}>
        {t('sidebar.brand')}
      </Text>
    </Group>
  );
}

function SpaceBox() {
  const { t } = useTranslation();
  const { tasks, statuses } = useSidebarCounts();

  return (
    <Box px={11} py={10} className={classes.spaceBox}>
      <Group gap={10} wrap="nowrap">
        <ThemeIcon size={28} radius={8} variant="light" color="brand">
          <IconChecklist size={16} />
        </ThemeIcon>
        <div className={classes.spaceBoxBody}>
          <Text fz={12.5} fw={700} c={tokens.text} truncate>
            {t('sidebar.spaceTitle')}
          </Text>
          <Text fz={11} c={tokens.textMuted} truncate>
            {t('sidebar.spaceMeta', { tasks, statuses })}
          </Text>
        </div>
      </Group>
    </Box>
  );
}

export function Sidebar({ onNavigate }: { onNavigate?: () => void }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { pathname } = useLocation();

  const select = (to: string) => {
    navigate(to);
    onNavigate?.();
  };

  return (
    <Stack gap={0} h="100%" className={classes.root}>
      <Box px="md" pt="md">
        <WorkspaceHeader />
        <Box mt={14}>
          <SpaceBox />
        </Box>
      </Box>

      <Box className={`${scrollClasses.scroll} ${classes.scrollArea}`} px="md" pb="md">
        <SectionLabel>{t('nav.mainMenu')}</SectionLabel>
        <Stack gap={2}>
          {MAIN_MENU.map((entry) => (
            <NavItem
              key={entry.labelKey}
              entry={entry}
              active={pathname === entry.to}
              onSelect={select}
            />
          ))}
        </Stack>
      </Box>
    </Stack>
  );
}

export default Sidebar;
