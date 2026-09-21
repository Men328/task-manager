import {
  ActionIcon,
  Avatar,
  Burger,
  Group,
  Indicator,
  Menu,
  Stack,
  Text,
  TextInput,
  UnstyledButton,
} from '@mantine/core';
import { IconBell, IconLogout, IconSearch } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { useSession } from '../../context';
import { avatarColor, initials } from '../../lib/format';
import { tokens } from '../../theme';
import { LanguageSwitcher } from './LanguageSwitcher';
import { ThemeSwitcher } from './ThemeSwitcher';
import classes from './TopBar.module.css';

export function TopBar({ navOpened, onToggleNav }: { navOpened: boolean; onToggleNav: () => void }) {
  const { t } = useTranslation();
  const { profile, query, setQuery, signOut } = useSession();

  const name = profile?.displayName ?? t('topbar.noProfile');
  const subtitle = profile?.email ?? t('topbar.identityOffline');

  return (
    <Group h="100%" px="md" gap="md" wrap="nowrap" justify="space-between">
      <Group gap="sm" wrap="nowrap" className={classes.searchGroup}>
        <Burger
          opened={navOpened}
          onClick={onToggleNav}
          hiddenFrom="sm"
          size="sm"
          aria-label={t('topbar.openMenu')}
        />

        <TextInput
          value={query}
          onChange={(event) => setQuery(event.currentTarget.value)}
          placeholder={t('topbar.searchPlaceholder')}
          leftSection={<IconSearch size={16} style={{ color: tokens.textFaint }} />}
          rightSection={
            <Text fz={11} fw={600} c={tokens.textFaint}>
              ⌘K
            </Text>
          }
          rightSectionWidth={40}
          radius="md"
          w={{ base: 200, sm: 320, md: 380 }}
          visibleFrom="xs"
          classNames={{ input: classes.searchInput }}
        />
      </Group>

      <Group gap="sm" wrap="nowrap">
        <ThemeSwitcher />
        <LanguageSwitcher />

        <Indicator color="red" size={7} offset={4} withBorder disabled={false}>
          <ActionIcon
            variant="subtle"
            color="gray"
            size="lg"
            radius="md"
            aria-label={t('topbar.notifications')}
          >
            <IconBell size={18} style={{ color: tokens.textMuted }} />
          </ActionIcon>
        </Indicator>

        <Menu shadow="md" width={220} position="bottom-end" radius="md" withinPortal>
          <Menu.Target>
            <UnstyledButton aria-label={t('topbar.signOut')}>
              <Group gap={9} wrap="nowrap" pl={4}>
                <Avatar size={34} radius={999} color={avatarColor(name)}>
                  {initials(name)}
                </Avatar>
                <Stack gap={0} visibleFrom="sm">
                  <Text fz={12.5} fw={700} c={tokens.text} truncate maw={150}>
                    {name}
                  </Text>
                  <Text fz={11} c={tokens.textMuted} truncate maw={150}>
                    {subtitle}
                  </Text>
                </Stack>
              </Group>
            </UnstyledButton>
          </Menu.Target>

          <Menu.Dropdown>
            <Menu.Label>
              <Text fz={11} truncate>
                {subtitle}
              </Text>
            </Menu.Label>
            <Menu.Item leftSection={<IconLogout size={15} />} color="red" onClick={signOut}>
              {t('topbar.signOut')}
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Group>
    </Group>
  );
}

export default TopBar;
