import { Box, Group, Stack, Text, ThemeIcon, UnstyledButton } from '@mantine/core';
import { IconSparkles } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router-dom';

import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import { WorkspaceSwitcher } from '../workspace/WorkspaceSwitcher';
import { NAV_SECTIONS, type NavItem } from './navigation';
import classes from './Sidebar.module.css';

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

function EmptySectionHint() {
  const { t } = useTranslation();

  return (
    <Text fz={12} c={tokens.textFaint} className={classes.emptyHint} px={10} py={7}>
      {t('nav.emptyGroup')}
    </Text>
  );
}

function NavItemRow({
  entry,
  active,
  onSelect,
}: {
  entry: NavItem;
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
          <WorkspaceSwitcher />
        </Box>
      </Box>

      <Box className={`${scrollClasses.scroll} ${classes.scrollArea}`} px="md" pb="md">
        {NAV_SECTIONS.map((section) => (
          <Box key={section.key}>
            <SectionLabel>{t(section.labelKey)}</SectionLabel>
            {section.items.length === 0 ? (
              <EmptySectionHint />
            ) : (
              <Stack gap={2}>
                {section.items.map((entry) => (
                  <NavItemRow
                    key={entry.key}
                    entry={entry}
                    active={pathname === entry.to}
                    onSelect={select}
                  />
                ))}
              </Stack>
            )}
          </Box>
        ))}
      </Box>
    </Stack>
  );
}

export default Sidebar;
