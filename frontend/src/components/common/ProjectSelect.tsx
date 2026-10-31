import { useMemo } from 'react';
import { Group, Select, Tooltip } from '@mantine/core';
import type { ComboboxItem } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { useProject } from '../../context';
import { projectColor } from '../../lib/tokens';
import { tokens } from '../../theme';

interface ProjectSelectProps {
  value: string;
  onChange: (id: string) => void;
  disabled?: boolean;
  required?: boolean;
  /** Tooltip hiển thị khi `disabled` (ví dụ subtask theo project của task cha). */
  disabledReason?: string;
}

function Dot({ color }: { color?: string | null }) {
  return (
    <span
      style={{
        width: 9,
        height: 9,
        borderRadius: 999,
        flex: '0 0 auto',
        background: color || tokens.brand,
      }}
    />
  );
}

/** Chọn project (chỉ project đang hoạt động), mỗi option có chấm màu. */
export function ProjectSelect({
  value,
  onChange,
  disabled,
  required,
  disabledReason,
}: ProjectSelectProps) {
  const { t } = useTranslation();
  const { projects, activeProjects } = useProject();

  const colorById = useMemo(() => {
    const map = new Map<string, string | null | undefined>();
    projects.forEach((project) => map.set(project.id, projectColor(project)));
    return map;
  }, [projects]);

  const data = useMemo(() => {
    const items = activeProjects.map((project) => ({ value: project.id, label: project.name }));
    // Item đang nằm trong project đã lưu trữ vẫn phải hiển thị được project hiện tại.
    if (value && !items.some((item) => item.value === value)) {
      const current = projects.find((project) => project.id === value);
      if (current) {
        items.unshift({ value: current.id, label: current.name });
      }
    }
    return items;
  }, [activeProjects, projects, value]);

  const select = (
    <Select
      label={t('project.selectLabel')}
      placeholder={t('project.selectPlaceholder')}
      data={data}
      value={value || null}
      onChange={(next) => {
        if (next) {
          onChange(next);
        }
      }}
      searchable
      allowDeselect={false}
      required={required}
      disabled={disabled}
      nothingFoundMessage={t('project.noMatch')}
      leftSection={value ? <Dot color={colorById.get(value)} /> : undefined}
      renderOption={({ option }: { option: ComboboxItem }) => (
        <Group gap={8} wrap="nowrap">
          <Dot color={colorById.get(option.value)} />
          <span>{option.label}</span>
        </Group>
      )}
    />
  );

  if (disabled && disabledReason) {
    return (
      <Tooltip label={disabledReason} withArrow>
        <div>{select}</div>
      </Tooltip>
    );
  }
  return select;
}

export default ProjectSelect;
