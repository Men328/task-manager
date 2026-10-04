import { useEffect, useRef, useState } from 'react';
import {
  ActionIcon,
  Box,
  Button,
  FileButton,
  Group,
  Loader,
  Stack,
  Text,
  Tooltip,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconDownload, IconFile, IconPaperclip, IconTrash, IconUpload } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import {
  attachmentDownloadUrl,
  deleteAttachment,
  listAttachments,
  uploadAttachment,
} from '../../api/attachment';
import { getErrorMessage } from '../../api/client';
import { useSession } from '../../context';
import { formatBytes } from '../../lib/format';
import { tokens } from '../../theme';
import type { Attachment, AttachmentOwnerType } from '../../types';
import classes from './AttachmentSection.module.css';

interface AttachmentSectionProps {
  ownerType: AttachmentOwnerType;
  /** Id đối tượng; rỗng = chưa lưu nên chưa thể đính kèm. */
  ownerId: string | null | undefined;
}

/**
 * Khối tệp đính kèm dùng chung cho task/lịch/sự kiện/backlog.
 * Chỉ hiển thị khi đối tượng đã có id (đã lưu).
 */
export function AttachmentSection({ ownerType, ownerId }: AttachmentSectionProps) {
  const { t } = useTranslation();
  const { profileId } = useSession();

  const [items, setItems] = useState<Attachment[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [removingId, setRemovingId] = useState<string | null>(null);
  const resetRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (!ownerId) {
      setItems([]);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    listAttachments(ownerType, ownerId)
      .then((list) => {
        if (!cancelled) {
          setItems(list);
        }
      })
      .catch((cause) => {
        if (!cancelled) {
          setError(getErrorMessage(cause));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [ownerType, ownerId]);

  const handleFile = async (file: File | null) => {
    if (!file) {
      return;
    }
    if (!profileId || !ownerId) {
      notifications.show({
        title: t('attachment.uploadFailed'),
        message: t('attachment.needProfile'),
        color: 'red',
      });
      return;
    }

    setUploading(true);
    setError(null);
    try {
      const created = await uploadAttachment({ profileId, ownerType, ownerId, file });
      setItems((prev) => [created, ...prev.filter((item) => item.id !== created.id)]);
      notifications.show({
        title: t('attachment.uploaded'),
        message: created.fileName,
        color: 'teal',
      });
    } catch (cause) {
      const message = getErrorMessage(cause);
      setError(message);
      notifications.show({ title: t('attachment.uploadFailed'), message, color: 'red' });
    } finally {
      setUploading(false);
      resetRef.current?.();
    }
  };

  const handleRemove = async (item: Attachment) => {
    setRemovingId(item.id);
    try {
      await deleteAttachment(item.id);
      setItems((prev) => prev.filter((entry) => entry.id !== item.id));
      notifications.show({
        title: t('attachment.removed'),
        message: item.fileName,
        color: 'teal',
      });
    } catch (cause) {
      notifications.show({
        title: t('attachment.removeFailed'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setRemovingId(null);
    }
  };

  return (
    <Stack gap={7}>
      <Group justify="space-between" wrap="nowrap">
        <Group gap={6} wrap="nowrap">
          <IconPaperclip size={14} style={{ color: tokens.textMuted }} />
          <Text fz={11} fw={700} c={tokens.textMuted} tt="uppercase">
            {t('attachment.title')}
          </Text>
          {items.length > 0 ? (
            <Text fz={11} c={tokens.textFaint}>
              ({items.length})
            </Text>
          ) : null}
        </Group>

        <FileButton resetRef={resetRef} onChange={handleFile}>
          {(props) => (
            <Button
              {...props}
              size="compact-xs"
              variant="light"
              leftSection={<IconUpload size={13} />}
              loading={uploading}
              disabled={!profileId || !ownerId}
            >
              {t('attachment.upload')}
            </Button>
          )}
        </FileButton>
      </Group>

      {loading ? (
        <Group gap={6} wrap="nowrap">
          <Loader size="xs" />
          <Text fz={12} c={tokens.textMuted}>
            {t('attachment.loading')}
          </Text>
        </Group>
      ) : null}

      {error ? (
        <Text fz={12} c={tokens.danger}>
          {error}
        </Text>
      ) : null}

      {!loading && items.length === 0 && !error ? (
        <Text fz={12} c={tokens.textFaint}>
          {t('attachment.empty')}
        </Text>
      ) : null}

      {items.map((item) => (
        <Group
          key={item.id}
          className={classes.row}
          justify="space-between"
          wrap="nowrap"
          gap="xs"
        >
          <Group gap={8} wrap="nowrap" style={{ minWidth: 0 }}>
            <IconFile size={16} style={{ color: tokens.textMuted, flex: '0 0 auto' }} />
            <Box style={{ minWidth: 0 }}>
              <Text fz={12.5} fw={600} c={tokens.text} truncate>
                {item.fileName}
              </Text>
              <Text fz={11} c={tokens.textFaint}>
                {formatBytes(item.size)}
              </Text>
            </Box>
          </Group>

          <Group gap={4} wrap="nowrap">
            <Tooltip label={t('attachment.download')} withArrow openDelay={300}>
              <ActionIcon
                component="a"
                href={attachmentDownloadUrl(item.id)}
                download={item.fileName}
                target="_blank"
                rel="noreferrer"
                variant="subtle"
                color="gray"
                size="sm"
                aria-label={t('attachment.downloadAria', { name: item.fileName })}
              >
                <IconDownload size={15} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label={t('common.remove')} withArrow openDelay={300}>
              <ActionIcon
                variant="subtle"
                color="red"
                size="sm"
                loading={removingId === item.id}
                onClick={() => void handleRemove(item)}
                aria-label={t('attachment.removeAria', { name: item.fileName })}
              >
                <IconTrash size={15} />
              </ActionIcon>
            </Tooltip>
          </Group>
        </Group>
      ))}
    </Stack>
  );
}

export default AttachmentSection;
