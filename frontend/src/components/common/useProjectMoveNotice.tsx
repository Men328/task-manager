import { useCallback } from 'react';
import { Button, Group, Text } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useTranslation } from 'react-i18next';

import { useProject } from '../../context';

/**
 * Sau khi lưu một item mà project của nó khác project đang chọn, item sẽ biến
 * khỏi trang hiện tại -> báo cho người dùng kèm nút chuyển sang project đó.
 */
export function useProjectMoveNotice() {
  const { t } = useTranslation();
  const { projects, currentProjectId, setCurrent } = useProject();

  return useCallback(
    (projectId: string) => {
      if (!projectId || projectId === currentProjectId) {
        return;
      }
      const name = projects.find((project) => project.id === projectId)?.name ?? '';
      const id = `project-move-${projectId}`;
      notifications.show({
        id,
        color: 'blue',
        autoClose: 8000,
        message: (
          <Group justify="space-between" wrap="nowrap" gap="sm">
            <Text fz={13}>{t('project.movedNotice', { name })}</Text>
            <Button
              size="compact-xs"
              variant="light"
              onClick={() => {
                setCurrent(projectId);
                notifications.hide(id);
              }}
            >
              {t('project.goTo')}
            </Button>
          </Group>
        ),
      });
    },
    [t, projects, currentProjectId, setCurrent],
  );
}
