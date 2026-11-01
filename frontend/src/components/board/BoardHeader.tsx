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
  IconArrowsSort,
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
  IconX,
} from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { useBoard, useSession } from '../../context';
import { useBoardFilters } from '../../context/board/useBoardFilters';
import { DUE_FILTERS, SORT_FIELDS, activeFilterCount } from '../../lib/boardFilters';
import type { DueFilter, SortField } from '../../lib/boardFilters';
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
  const { refresh, loading } = useBoard();
  const { filters, update, clear } = useBoardFilters();

  const filterCount = activeFilterCount(filters);
  const togglePriority = (priority: (typeof TASK_PRIORITIES)[number]) =>
    update({
      priorities: filters.priorities.includes(priority)
        ? filters.priorities.filter((item) => item !== priority)
        : [...filters.priorities, priority],
    });
  const clearAll = () => {
    clear();
    setQuery('');
  };
  const check = (on: boolean) =>
    on ? <IconCheck size={14} style={{ color: tokens.brand }} /> : <Box w={14} />;
  const dueLabel = (due: DueFilter) => t(`boardHeader.due_${due}`);
  const sortLabel = (field: SortField) => t(`boardHeader.sort_${field}`);

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
              <Menu.Label>{t('boardHeader.due')}</Menu.Label>
              {DUE_FILTERS.map((due) => (
                <Menu.Item key={due} onClick={() => update({ due })} leftSection={check(filters.due === due)}>
                  {dueLabel(due)}
                </Menu.Item>
              ))}
              <Menu.Divider />
              <Menu.Label>{t('boardHeader.priority')}</Menu.Label>
              {TASK_PRIORITIES.map((priority) => (
                <Menu.Item
                  key={priority}
                  onClick={() => togglePriority(priority)}
                  leftSection={check(filters.priorities.includes(priority))}
                >
                  {t(PRIORITY_LABEL_KEY[priority])}
                </Menu.Item>
              ))}
              <Menu.Divider />
              <Menu.Item
                onClick={() => update({ hideDone: !filters.hideDone })}
                leftSection={check(filters.hideDone)}
              >
                {t('boardHeader.hideDone')}
              </Menu.Item>
              <Menu.Divider />
              <Menu.Item leftSection={<IconRefresh size={14} />} onClick={clearAll}>
                {t('boardHeader.clearFilter')}
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>

          {view === 'table' ? (
            <Menu position="bottom-end" withArrow shadow="md">
              <Menu.Target>
                <Button
                  variant="default"
                  size="compact-sm"
                  leftSection={<IconArrowsSort size={14} />}
                  rightSection={<IconChevronDown size={13} />}
                >
                  {`${sortLabel(filters.sort.field)} ${filters.sort.desc ? '↓' : '↑'}`}
                </Button>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>{t('boardHeader.sortBy')}</Menu.Label>
                {SORT_FIELDS.map((field) => (
                  <Menu.Item
                    key={field}
                    onClick={() => update({ sort: { ...filters.sort, field } })}
                    leftSection={check(filters.sort.field === field)}
                  >
                    {sortLabel(field)}
                  </Menu.Item>
                ))}
                <Menu.Divider />
                <Menu.Item
                  onClick={() => update({ sort: { ...filters.sort, desc: false } })}
                  leftSection={check(!filters.sort.desc)}
                >
                  {t('boardHeader.sortAsc')}
                </Menu.Item>
                <Menu.Item
                  onClick={() => update({ sort: { ...filters.sort, desc: true } })}
                  leftSection={check(filters.sort.desc)}
                >
                  {t('boardHeader.sortDesc')}
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          ) : null}

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

      {filterCount > 0 ? (
        <Group gap={6} pb="sm" wrap="wrap">
          {filters.due !== 'all' ? (
            <FilterChip label={dueLabel(filters.due)} onRemove={() => update({ due: 'all' })} />
          ) : null}
          {filters.priorities.map((priority) => (
            <FilterChip
              key={priority}
              label={t(PRIORITY_LABEL_KEY[priority])}
              onRemove={() => togglePriority(priority)}
            />
          ))}
          {filters.hideDone ? (
            <FilterChip label={t('boardHeader.hideDone')} onRemove={() => update({ hideDone: false })} />
          ) : null}
          <Button size="compact-xs" variant="subtle" onClick={clearAll}>
            {t('boardHeader.clearFilter')}
          </Button>
        </Group>
      ) : null}
    </Box>
  );
}

function FilterChip({ label, onRemove }: { label: string; onRemove: () => void }) {
  const { t } = useTranslation();
  return (
    <Group gap={4} wrap="nowrap" className={classes.filterChip}>
      <Text fz={12} fw={600} c={tokens.text}>
        {label}
      </Text>
      <ActionIcon
        size="xs"
        variant="subtle"
        color="gray"
        radius="xl"
        onClick={onRemove}
        aria-label={t('boardHeader.removeFilter', { label })}
      >
        <IconX size={12} />
      </ActionIcon>
    </Group>
  );
}

export default BoardHeader;
