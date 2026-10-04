import {
  ActionIcon,
  Button,
  Center,
  Group,
  Indicator,
  Loader,
  Popover,
  ScrollArea,
  Stack,
  Text,
  UnstyledButton,
} from '@mantine/core';
import { IconBell, IconBellOff, IconCheck } from '@tabler/icons-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { useNotices } from '../../context';
import { formatDateTime } from '../../lib/format';
import { tokens } from '../../theme';
import { normalizeTargetType, type Notice } from '../../types';
import classes from './NotificationMenu.module.css';

function targetPath(notice: Notice): string {
  switch (normalizeTargetType(notice.targetType)) {
    case 'schedule':
      return '/calendar';
    case 'event':
      return '/events';
    case 'backlog':
      return '/backlog';
    default:
      return '/';
  }
}

export function NotificationMenu() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { items, unreadCount, loading, markRead, markAllRead, refresh } = useNotices();
  const [opened, setOpened] = useState(false);

  const badge = unreadCount > 99 ? '99+' : String(unreadCount);

  const handleOpenedChange = (next: boolean) => {
    setOpened(next);
    if (next) {
      refresh();
    }
  };

  const openNotice = (notice: Notice) => {
    setOpened(false);
    if (!notice.isRead) {
      void markRead([notice.id]);
    }
    navigate(`${targetPath(notice)}?focus=${encodeURIComponent(notice.targetId)}`);
  };

  return (
    <Popover
      opened={opened}
      onChange={handleOpenedChange}
      width={360}
      position="bottom-end"
      shadow="md"
      radius="md"
      withinPortal
    >
      <Popover.Target>
        <Indicator
          color="red"
          size={18}
          offset={6}
          disabled={unreadCount === 0}
          label={badge}
          classNames={{ indicator: classes.indicator }}
          onClick={() => handleOpenedChange(!opened)}
        >
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
      </Popover.Target>

      <Popover.Dropdown p={0}>
        <Group justify="space-between" px="md" py="sm" className={classes.header}>
          <Text fz={13} fw={700} c={tokens.text}>
            {t('notice.title')}
          </Text>
          <Button
            variant="subtle"
            size="compact-xs"
            leftSection={<IconCheck size={13} />}
            disabled={unreadCount === 0}
            onClick={() => void markAllRead()}
          >
            {t('notice.markAll')}
          </Button>
        </Group>

        <ScrollArea.Autosize mah={360} type="hover">
          {loading && items.length === 0 ? (
            <Center py="lg">
              <Loader size="sm" />
            </Center>
          ) : items.length === 0 ? (
            <Stack align="center" gap={6} py="xl" px="md">
              <IconBellOff size={22} style={{ color: tokens.textFaint }} />
              <Text fz={12.5} c={tokens.textMuted}>
                {t('notice.empty')}
              </Text>
            </Stack>
          ) : (
            <Stack gap={0}>
              {items.map((notice) => (
                <UnstyledButton
                  key={notice.id}
                  className={classes.item}
                  onClick={() => openNotice(notice)}
                >
                  <Group justify="space-between" gap="xs" wrap="nowrap" align="flex-start">
                    <Text
                      fz={13}
                      fw={notice.isRead ? 600 : 700}
                      c={tokens.text}
                      className={classes.itemTitle}
                    >
                      {notice.title}
                    </Text>
                    {notice.isRead ? null : <span className={classes.dot} />}
                  </Group>
                  {notice.body ? (
                    <Text fz={12} c={tokens.textMuted} lineClamp={2} className={classes.itemBody}>
                      {notice.body}
                    </Text>
                  ) : null}
                  <Text fz={11} c={tokens.textFaint}>
                    {formatDateTime(notice.createdAt) ?? ''}
                  </Text>
                </UnstyledButton>
              ))}
            </Stack>
          )}
        </ScrollArea.Autosize>
      </Popover.Dropdown>
    </Popover>
  );
}

export default NotificationMenu;
