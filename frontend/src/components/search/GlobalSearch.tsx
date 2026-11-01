import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { ActionIcon, Combobox, Loader, Modal, Text, TextInput, UnstyledButton, useCombobox } from '@mantine/core';
import { IconSearch } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { getTask } from '../../api/task';
import { useAppDispatch, useProject, useSession } from '../../context';
import { fetchBoard } from '../../context/board/boardSlice';
import { tokens } from '../../theme';
import type { Task } from '../../types';
import TaskDetailModal from '../task/TaskDetailModal';
import classes from './GlobalSearch.module.css';
import SearchResultRow from './SearchResultRow';
import { MIN_QUERY_LENGTH, useTaskSearch } from './useTaskSearch';
import type { SearchState } from './useTaskSearch';

const IS_MAC = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform);

/**
 * Ô tìm kiếm task của mọi project trên TopBar: dropdown kết quả (↑/↓/Enter), ⌘K / Ctrl+K để
 * focus, và trên mobile là nút kính lúp mở màn tìm kiếm toàn màn hình.
 */
export function GlobalSearch() {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { profileId } = useSession();
  const { projects, setCurrent } = useProject();
  const [query, setQuery] = useState('');
  const [mobileOpened, setMobileOpened] = useState(false);
  const [detailTask, setDetailTask] = useState<Task | null>(null);
  const [detailOpened, setDetailOpened] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const combobox = useCombobox({ onDropdownClose: () => combobox.resetSelectedOption() });
  const { state, statuses } = useTaskSearch(profileId, query);
  const projectById = new Map(projects.map((project) => [project.id, project]));

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        if (inputRef.current && inputRef.current.offsetParent !== null) {
          inputRef.current.focus();
          inputRef.current.select();
          combobox.openDropdown();
        } else {
          setMobileOpened(true);
        }
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [combobox]);

  useEffect(() => {
    if (state.kind === 'done' && state.tasks.length > 0) {
      combobox.selectFirstOption();
    }
  }, [state, combobox]);

  const tasks = state.kind === 'done' ? state.tasks : [];

  // Mở task giống màn Hôm nay: chuyển switcher sang project của task và nạp board để modal có status/task cha.
  const openTask = async (task: Task) => {
    combobox.closeDropdown();
    setMobileOpened(false);
    setCurrent(task.projectId);
    if (profileId) {
      void dispatch(fetchBoard({ profileId, projectId: task.projectId }));
    }
    setDetailTask(task);
    setDetailOpened(true);
    const full = await getTask(task.id).catch(() => null);
    if (full) {
      setDetailTask(full);
    }
  };

  const message = (current: SearchState): ReactNode => {
    switch (current.kind) {
      case 'tooShort':
        return t('search.minChars', { count: MIN_QUERY_LENGTH });
      case 'loading':
        return t('search.loading');
      case 'error':
        return current.message;
      case 'done':
        return current.tasks.length === 0 ? t('search.noResults') : null;
      default:
        return null;
    }
  };
  const status = message(state);

  const row = (task: Task) => (
    <SearchResultRow
      task={task}
      query={query}
      project={projectById.get(task.projectId)}
      status={statuses.get(task.statusId)}
    />
  );

  return (
    <>
      <Combobox
        store={combobox}
        onOptionSubmit={(id) => {
          const task = tasks.find((item) => item.id === id);
          if (task) {
            void openTask(task);
          }
        }}
        width="target"
        position="bottom-start"
        shadow="md"
        withinPortal
        classNames={{ option: classes.option }}
      >
        <Combobox.Target>
          <TextInput
            ref={inputRef}
            value={query}
            onChange={(event) => {
              setQuery(event.currentTarget.value);
              combobox.openDropdown();
            }}
            onFocus={() => combobox.openDropdown()}
            onBlur={() => combobox.closeDropdown()}
            onKeyDown={(event) => {
              if (event.key === 'Escape') {
                combobox.closeDropdown();
                event.currentTarget.blur();
              }
            }}
            placeholder={t('topbar.searchPlaceholder')}
            aria-label={t('topbar.searchPlaceholder')}
            leftSection={<IconSearch size={16} style={{ color: tokens.textFaint }} />}
            rightSection={
              state.kind === 'loading' ? (
                <Loader size={14} />
              ) : (
                <Text fz={11} fw={600} c={tokens.textFaint}>
                  {IS_MAC ? '⌘K' : 'Ctrl K'}
                </Text>
              )
            }
            rightSectionWidth={52}
            radius="md"
            w={{ base: 200, sm: 320, md: 380 }}
            visibleFrom="xs"
            classNames={{ input: classes.searchInput }}
          />
        </Combobox.Target>

        <Combobox.Dropdown hidden={state.kind === 'idle'}>
          <Combobox.Options mah={420} style={{ overflowY: 'auto' }}>
            {tasks.map((task) => (
              <Combobox.Option key={task.id} value={task.id}>
                {row(task)}
              </Combobox.Option>
            ))}
            {status ? (
              <Text fz={12.5} c={tokens.textMuted} className={classes.message}>
                {status}
              </Text>
            ) : null}
          </Combobox.Options>
        </Combobox.Dropdown>
      </Combobox>

      <ActionIcon
        variant="default"
        size="lg"
        radius="md"
        hiddenFrom="xs"
        aria-label={t('topbar.searchPlaceholder')}
        onClick={() => setMobileOpened(true)}
      >
        <IconSearch size={16} />
      </ActionIcon>

      <Modal
        opened={mobileOpened}
        onClose={() => setMobileOpened(false)}
        fullScreen
        title={t('search.title')}
        radius={0}
      >
        <TextInput
          data-autofocus
          value={query}
          onChange={(event) => setQuery(event.currentTarget.value)}
          placeholder={t('topbar.searchPlaceholder')}
          leftSection={<IconSearch size={16} style={{ color: tokens.textFaint }} />}
          radius="md"
          classNames={{ input: classes.searchInput }}
          mb="sm"
        />
        <div className={classes.mobileList}>
          {tasks.map((task) => (
            <UnstyledButton key={task.id} className={classes.mobileItem} onClick={() => void openTask(task)}>
              {row(task)}
            </UnstyledButton>
          ))}
        </div>
        {status ? (
          <Text fz={12.5} c={tokens.textMuted} className={classes.message}>
            {status}
          </Text>
        ) : null}
      </Modal>

      <TaskDetailModal opened={detailOpened} onClose={() => setDetailOpened(false)} task={detailTask} />
    </>
  );
}

export default GlobalSearch;
