import { Group, Menu, Text, UnstyledButton } from '@mantine/core';
import { IconCheck, IconChevronDown, IconWorld } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { useLanguage } from '../../i18n/useLanguage';
import classes from './LanguageSwitcher.module.css';

export function LanguageSwitcher() {
  const { t } = useTranslation();
  const { language, languages, changeLanguage } = useLanguage();
  const active = languages.find((item) => item.code === language) ?? languages[0];

  return (
    <Menu position="bottom-end" withArrow shadow="md">
      <Menu.Target>
        <UnstyledButton className={classes.trigger} aria-label={t('topbar.language')}>
          <Group gap={5} wrap="nowrap">
            <IconWorld size={15} />
            <Text fz={13} fw={600}>
              {active.short}
            </Text>
            <IconChevronDown size={14} />
          </Group>
        </UnstyledButton>
      </Menu.Target>
      <Menu.Dropdown>
        <Menu.Label>{t('topbar.language')}</Menu.Label>
        {languages.map((item) => (
          <Menu.Item
            key={item.code}
            onClick={() => changeLanguage(item.code)}
            leftSection={
              item.code === language ? (
                <IconCheck size={14} className={classes.check} />
              ) : (
                <span className={classes.checkPlaceholder} />
              )
            }
          >
            {item.label}
          </Menu.Item>
        ))}
      </Menu.Dropdown>
    </Menu>
  );
}

export default LanguageSwitcher;
