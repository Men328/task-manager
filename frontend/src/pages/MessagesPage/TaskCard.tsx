import { useEffect, useState } from 'react';
import { Badge, Box, Button, Group, Stack, Text, UnstyledButton } from '@mantine/core';
import {
  IconCircleCheck,
  IconCircleDashed,
  IconCircleX,
  IconDownload,
  IconPaperclip,
  IconProgress,
} from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { attachmentDownloadUrl, listAttachments } from '../../api/attachment';
import { formatBytes } from '../../lib/format';
import { isRecentTask, RECENT_TASK_RETRY_DELAYS_MS, subscribeTaskAttachments } from '../../lib/taskAttachments';
import { tokens } from '../../theme';
import type { Attachment, ChatTaskCategory, ChatTaskRef } from '../../types';
import classes from './MessagesPage.module.css';

const CATEGORY_COLOR: Record<string, string> = {
  todo: 'gray',
  in_progress: 'blue',
  done: 'green',
  cancelled: 'red',
};

function CategoryIcon({ category }: { category: ChatTaskCategory }) {
  const props = { size: 18, stroke: 1.8 };
  switch (category) {
    case 'done':
      return <IconCircleCheck {...props} color="var(--mantine-color-green-6)" />;
    case 'cancelled':
      return <IconCircleX {...props} color="var(--mantine-color-red-6)" />;
    case 'in_progress':
      return <IconProgress {...props} color="var(--mantine-color-blue-6)" />;
    default:
      return <IconCircleDashed {...props} color="var(--mantine-color-gray-6)" />;
  }
}

/**
 * Danh sách tệp đính kèm của task giao trong chat. Tải lại khi người gửi vừa tải tệp
 * (bus trong cùng trình duyệt) và, với tin vừa gửi, thử lại vài lần để phía người nhận
 * thấy tệp được tải lên ngay sau khi task tới.
 */
function useTaskFiles(taskId: string, createdAt: string | undefined) {
  const [items, setItems] = useState<Attachment[]>([]);

  useEffect(() => {
    if (!taskId) {
      return;
    }
    let cancelled = false;
    const load = () => {
      listAttachments('task', taskId)
        .then((list) => {
          if (!cancelled) {
            setItems(list);
          }
        })
        .catch(() => {
          // Thẻ task vẫn hiển thị khi không đọc được tệp; khung chi tiết task sẽ báo lỗi.
        });
    };

    load();
    const unsubscribe = subscribeTaskAttachments(taskId, load);
    const timers = isRecentTask(createdAt) ? RECENT_TASK_RETRY_DELAYS_MS.map((delay) => window.setTimeout(load, delay)) : [];

    return () => {
      cancelled = true;
      unsubscribe();
      timers.forEach((timer) => window.clearTimeout(timer));
    };
  }, [taskId, createdAt]);

  return items;
}

/** Thẻ task trong tin nhắn; chỉ người nhận (chủ task) mở được task. */
export function TaskCard({
  task,
  mine,
  peerName,
  createdAt,
}: {
  task: ChatTaskRef;
  mine: boolean;
  peerName: string;
  createdAt?: string;
}) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const files = useTaskFiles(task.taskId, createdAt);

  return (
    <Box className={classes.taskCard} data-category={task.statusCategory || 'todo'}>
      <Text fz={11} fw={700} tt="uppercase" c={tokens.textFaint} mb={4}>
        {mine ? t('chatTask.assignedTo', { name: peerName }) : t('chatTask.assignedBy', { name: peerName })}
      </Text>
      <Group gap={8} wrap="nowrap" align="flex-start">
        <Box pt={1}>
          <CategoryIcon category={task.statusCategory} />
        </Box>
        <Text
          fz={14}
          fw={600}
          c={tokens.text}
          className={classes.bubbleText}
          td={task.statusCategory === 'done' || task.statusCategory === 'cancelled' ? 'line-through' : undefined}
        >
          {task.title}
        </Text>
      </Group>
      {files.length > 0 ? (
        <Stack gap={4} mt={8}>
          <Group gap={5} wrap="nowrap">
            <IconPaperclip size={13} style={{ color: tokens.textMuted }} />
            <Text fz={11} fw={700} tt="uppercase" c={tokens.textMuted}>
              {t('chatTask.filesCount', { count: files.length })}
            </Text>
          </Group>
          {files.map((file) => (
            <UnstyledButton
              key={file.id}
              component="a"
              href={attachmentDownloadUrl(file.id)}
              download={file.fileName}
              target="_blank"
              rel="noreferrer"
              className={classes.taskFile}
              aria-label={t('attachment.downloadAria', { name: file.fileName })}
            >
              <Group gap={6} wrap="nowrap" justify="space-between">
                <Text fz={12.5} fw={600} c={tokens.text} truncate>
                  {file.fileName}
                </Text>
                <Group gap={4} wrap="nowrap" style={{ flex: '0 0 auto' }}>
                  <Text fz={11} c={tokens.textFaint}>
                    {formatBytes(file.size)}
                  </Text>
                  <IconDownload size={13} style={{ color: tokens.textMuted }} />
                </Group>
              </Group>
            </UnstyledButton>
          ))}
        </Stack>
      ) : null}
      <Group justify="space-between" mt={8} gap="xs">
        <Badge variant="light" color={CATEGORY_COLOR[task.statusCategory] ?? 'gray'} radius="sm">
          {task.statusName || t('chatTask.unknownStatus')}
        </Badge>
        {!mine ? (
          <Button size="compact-xs" variant="subtle" onClick={() => navigate(`/?focus=${encodeURIComponent(task.taskId)}`)}>
            {t('chatTask.open')}
          </Button>
        ) : null}
      </Group>
    </Box>
  );
}
