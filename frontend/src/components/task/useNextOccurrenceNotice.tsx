import { Button, Group, Text } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconRepeat } from '@tabler/icons-react';
import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';

import { messageForCode } from '../../lib/errorCatalog';
import type { StatusChangeResult, Task } from '../../types';

/** Toast sau khi hoàn thành task lặp: đã tạo lần kế tiếp (có nút Mở) hoặc cảnh báo không tạo được. */
export function useNextOccurrenceNotice() {
  const { t, i18n } = useTranslation();

  return useCallback(
    (result: StatusChangeResult | undefined | null, onOpen?: (task: Task) => void) => {
      if (!result) {
        return;
      }
      const next = result.nextTask;
      if (next) {
        const due = next.dueAt
          ? new Date(next.dueAt).toLocaleDateString(i18n.language, { weekday: 'long', day: 'numeric', month: 'numeric' })
          : '';
        const id = `next-occurrence-${next.id}`;
        notifications.show({
          id,
          color: 'brand',
          icon: <IconRepeat size={16} />,
          autoClose: 8000,
          message: (
            <Group justify="space-between" wrap="nowrap" gap="sm">
              <Text fz={13}>{t('repeat.nextCreated', { date: due })}</Text>
              {onOpen ? (
                <Button
                  size="compact-xs"
                  variant="light"
                  onClick={() => {
                    notifications.hide(id);
                    onOpen(next);
                  }}
                >
                  {t('repeat.open')}
                </Button>
              ) : null}
            </Group>
          ),
        });
      } else if (result.nextTaskError) {
        notifications.show({
          color: 'yellow',
          title: t('repeat.nextFailed'),
          message: messageForCode(result.nextTaskError, i18n.language) ?? result.nextTaskError,
          autoClose: 10000,
        });
      }
    },
    [t, i18n.language],
  );
}
