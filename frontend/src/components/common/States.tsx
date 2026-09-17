import { Alert, Box, Button, Center, Code, Group, Loader, Stack, Text, ThemeIcon } from '@mantine/core';
import { IconAlertTriangle, IconMoodEmpty, IconRefresh } from '@tabler/icons-react';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { tokens } from '../../theme';
import classes from './States.module.css';

type IconComponent = typeof IconMoodEmpty;

/** Lỗi gọi API (backend chưa chạy / lỗi mạng). */
export function ApiErrorAlert({ message, onRetry }: { message: string; onRetry: () => void }) {
  const { t } = useTranslation();

  return (
    <Alert
      color="red"
      variant="light"
      icon={<IconAlertTriangle size={18} />}
      title={t('states.apiTitle')}
      radius="md"
    >
      <Stack gap="xs">
        <Text fz={13}>{message}</Text>
        <Text fz={12} c={tokens.textMuted}>
          {t('states.apiHint')}
        </Text>
        <Code>{'make run-identity   # :8081\nmake run-task       # :8082'}</Code>
        <Group>
          <Button
            size="xs"
            variant="light"
            color="red"
            leftSection={<IconRefresh size={14} />}
            onClick={onRetry}
          >
            {t('common.retry')}
          </Button>
        </Group>
      </Stack>
    </Alert>
  );
}

/** Panel rỗng ở giữa màn hình (chưa có profile / chưa có status). */
export function CenteredPanel({
  icon: Icon = IconMoodEmpty,
  title,
  description,
  action,
}: {
  icon?: IconComponent;
  title: string;
  description: ReactNode;
  action?: ReactNode;
}) {
  return (
    <Center className={classes.centered}>
      <Box p="xl" w={460} className={classes.panel}>
        <ThemeIcon size={46} radius={999} variant="light" color="brand" mx="auto">
          <Icon size={22} />
        </ThemeIcon>
        <Text fz={15.5} fw={800} c={tokens.text} mt="md">
          {title}
        </Text>
        <Text fz={13} c={tokens.textMuted} mt={6} lh={1.6}>
          {description}
        </Text>
        {action ? (
          <Group justify="center" mt="lg">
            {action}
          </Group>
        ) : null}
      </Box>
    </Center>
  );
}

export function LoadingBlock() {
  const { t } = useTranslation();

  return (
    <Center className={classes.centered}>
      <Stack align="center" gap="xs">
        <Loader size="sm" color="brand" />
        <Text fz={12.5} c={tokens.textMuted}>
          {t('states.loading')}
        </Text>
      </Stack>
    </Center>
  );
}
