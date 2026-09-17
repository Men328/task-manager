import {
  ActionIcon,
  Box,
  Button,
  Group,
  Menu,
  Text,
  TextInput,
  Title,
  Tooltip,
  UnstyledButton,
} from '@mantine/core';
import {
  IconChartHistogram,
  IconCheck,
  IconChevronDown,
  IconExternalLink,
  IconFilter,
  IconLayoutKanban,
  IconList,
  IconPlus,
  IconRefresh,
  IconSearch,
  IconTable,
} from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { useBoard, useSession } from '../../context';
import { PRIORITY_LABEL_KEY } from '../../lib/tokens';
import { tokens } from '../../theme';
import { TASK_PRIORITIES } from '../../types';
import classes from './BoardHeader.module.css';

export type BoardView = 'kanban' | 'table' | 'list' | 'timeline';

type IconComponent = typeof IconLayoutKanban;

const VIEWS: { value: BoardView; labelKey: string; icon: IconComponent; soon?: boolean }[] = [
  { value: 'kanban', labelKey: 'boardHeader.viewKanban', icon: IconLayoutKanban },
  { value: 'table', labelKey: 'boardHeader.viewTable', icon: IconTable },
  { value: 'list', labelKey: 'boardHeader.viewList', icon: IconList, soon: true },
  {
    value: 'timeline',
    labelKey: 'boardHeader.viewTimeline',
    icon: IconChartHistogram,
    soon: true,
  },
];

function ViewTab({
  label,
  icon: Icon,
  active,
  soon,
  onClick,
}: {
  label: string;
  icon: IconComponent;
  active: boolean;
  soon?: boolean;
  onClick: () => void;
}) {
  const { t } = useTranslation();

  const button = (
    <UnstyledButton
      onClick={onClick}
      disabled={soon}
      className={classes.viewTab}
      data-active={active}
      data-disabled={soon}
    >
      <Group gap={7} wrap="nowrap">
        <Icon size={15} stroke={1.9} style={{ color: active ? tokens.brand : tokens.textMuted }} />
        <Text fz={13} fw={active ? 700 : 600} c={active ? tokens.brand : tokens.navText}>
          {label}
        </Text>
      </Group>
    </UnstyledButton>
  );

  return soon ? (
    <Tooltip label={t('common.soon')} withArrow openDelay={250}>
      <Box>{button}</Box>
    </Tooltip>
  ) : (
    button
  );
}

interface BoardHeaderProps {
  view: BoardView;
  onViewChange: (view: BoardView) => void;
  onNewTask: () => void;
}

export function BoardHeader({ view, onViewChange, onNewTask }: BoardHeaderProps) {
  const { t } = useTranslation();
  const { query, setQuery } = useSession();
  const { priorityFilter, togglePriority, clearFilters, refresh, loading } = useBoard();

  const filterCount = priorityFilter.length;

  return (
    <Box px="lg" pt="md" className={classes.root}>
      <Group gap={7} wrap="nowrap">
        <Title order={3} fz={21} fw={800} c={tokens.text}>
          {t('boardHeader.title')}
        </Title>
        <IconExternalLink size={15} style={{ color: tokens.textFaint }} />
      </Group>

      <Group justify="space-between" align="center" gap="sm" mt={10} wrap="wrap">
        <Group gap={2} wrap="nowrap">
          {VIEWS.map((item) => (
            <ViewTab
              key={item.value}
              label={t(item.labelKey)}
              icon={item.icon}
              active={view === item.value}
              soon={item.soon}
              onClick={() => onViewChange(item.value)}
            />
          ))}
        </Group>

        <Group gap="xs" wrap="nowrap" pb="sm">
          <TextInput
            value={query}
            onChange={(event) => setQuery(event.currentTarget.value)}
            placeholder={t('boardHeader.searchPlaceholder')}
            leftSection={<IconSearch size={14} style={{ color: tokens.textFaint }} />}
            radius="md"
            size="sm"
            w={190}
            classNames={{ input: classes.searchInput }}
          />

          <Menu closeOnItemClick={false} position="bottom-end" withArrow shadow="md">
            <Menu.Target>
              <Button
                variant="default"
                size="compact-sm"
                leftSection={<IconFilter size={14} />}
                rightSection={<IconChevronDown size={13} />}
              >
                {filterCount > 0
                  ? t('boardHeader.filterCount', { n: filterCount })
                  : t('boardHeader.filter')}
              </Button>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Label>{t('boardHeader.priority')}</Menu.Label>
              {TASK_PRIORITIES.map((priority) => (
                <Menu.Item
                  key={priority}
                  onClick={() => togglePriority(priority)}
                  leftSection={
                    priorityFilter.includes(priority) ? (
                      <IconCheck size={14} style={{ color: tokens.brand }} />
                    ) : (
                      <Box w={14} />
                    )
                  }
                >
                  {t(PRIORITY_LABEL_KEY[priority])}
                </Menu.Item>
              ))}
              <Menu.Divider />
              <Menu.Item leftSection={<IconRefresh size={14} />} onClick={clearFilters}>
                {t('boardHeader.clearFilter')}
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>

          <Tooltip label={t('boardHeader.refresh')} withArrow openDelay={300}>
            <ActionIcon
              variant="default"
              size="lg"
              radius="md"
              onClick={refresh}
              loading={loading}
              aria-label={t('boardHeader.refreshAria')}
            >
              <IconRefresh size={15} />
            </ActionIcon>
          </Tooltip>

          <Button size="compact-sm" leftSection={<IconPlus size={15} />} onClick={onNewTask}>
            {t('boardHeader.newTask')}
          </Button>
        </Group>
      </Group>
    </Box>
  );
}

export default BoardHeader;
