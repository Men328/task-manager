import { Button, Group, Text } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconMessage } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router-dom';

import { useAppSelector, useChatRealtime } from '../../context';
import { tokens } from '../../theme';

const PREVIEW_LENGTH = 120;

/** Luôn mount trong layout: nhận tin realtime và báo khi người dùng không mở đúng cuộc trò chuyện. */
export function ChatRealtime() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const friends = useAppSelector((state) => state.friends.friends);

  useChatRealtime((message) => {
    const threadPath = `/messages/${message.senderId}`;
    if (pathname === threadPath) {
      return;
    }
    const sender = friends.find((friend) => friend.profile.id === message.senderId)?.profile;
    const toastId = `chat-${message.id}`;
    notifications.show({
      id: toastId,
      title: sender ? t('chat.newMessageFrom', { name: sender.displayName }) : t('chat.newMessage'),
      color: 'brand',
      icon: <IconMessage size={16} />,
      message: (
        <Group justify="space-between" wrap="nowrap" gap="sm">
          <Text fz={12.5} c={tokens.textMuted} lineClamp={2}>
            {message.body.slice(0, PREVIEW_LENGTH)}
          </Text>
          <Button
            size="compact-xs"
            variant="light"
            onClick={() => {
              notifications.hide(toastId);
              navigate(threadPath);
            }}
          >
            {t('chat.open')}
          </Button>
        </Group>
      ),
    });
  });

  return null;
}
