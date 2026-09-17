import { Menu, UnstyledButton, useMantineColorScheme } from '@mantine/core';
import { IconCheck, IconDeviceDesktop, IconMoon, IconSun } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import type { AppColorScheme } from '../../theme';
import classes from './ThemeSwitcher.module.css';

type IconComponent = typeof IconSun;

interface ThemeMode {
  value: AppColorScheme;
  labelKey: string;
  icon: IconComponent;
}

const MODES: ThemeMode[] = [
  { value: 'light', labelKey: 'theme.light', icon: IconSun },
  { value: 'dark', labelKey: 'theme.dark', icon: IconMoon },
  { value: 'auto', labelKey: 'theme.auto', icon: IconDeviceDesktop },
];

/**
 * Chọn chế độ sáng / tối / theo hệ thống. Giá trị lưu ở localStorage
 * (xem `COLOR_SCHEME_STORAGE_KEY` trong `src/theme.ts`).
 */
export function ThemeSwitcher() {
  const { t } = useTranslation();
  const { colorScheme, setColorScheme } = useMantineColorScheme();

  const active = MODES.find((mode) => mode.value === colorScheme) ?? MODES[2]!;
  const ActiveIcon = active.icon;

  return (
    <Menu position="bottom-end" withArrow shadow="md">
      <Menu.Target>
        <UnstyledButton className={classes.trigger} aria-label={t('topbar.theme')}>
          <ActiveIcon size={17} />
        </UnstyledButton>
      </Menu.Target>
      <Menu.Dropdown>
        <Menu.Label>{t('topbar.theme')}</Menu.Label>
        {MODES.map((mode) => (
          <Menu.Item
            key={mode.value}
            onClick={() => setColorScheme(mode.value)}
            leftSection={
              mode.value === colorScheme ? (
                <IconCheck size={14} className={classes.check} />
              ) : (
                <span className={classes.checkPlaceholder} />
              )
            }
          >
            {t(mode.labelKey)}
          </Menu.Item>
        ))}
      </Menu.Dropdown>
    </Menu>
  );
}

export default ThemeSwitcher;
