import { createStatus, createTransition } from '../api/task';
import type { TaskStatusCategory } from '../types';

interface SeedStatus {
  name: string;
  slug: string;
  color: string;
  category: TaskStatusCategory;
  isDefault: boolean;
  isTerminal: boolean;
}

const DEFAULT_STATUSES: SeedStatus[] = [
  {
    name: 'To Do',
    slug: 'todo',
    color: '#9aa0ae',
    category: 'TASK_STATUS_CATEGORY_TODO',
    isDefault: true,
    isTerminal: false,
  },
  {
    name: 'On Progress',
    slug: 'on-progress',
    color: '#f59f00',
    category: 'TASK_STATUS_CATEGORY_IN_PROGRESS',
    isDefault: false,
    isTerminal: false,
  },
  {
    name: 'In Review',
    slug: 'in-review',
    color: '#37b24d',
    category: 'TASK_STATUS_CATEGORY_IN_PROGRESS',
    isDefault: false,
    isTerminal: false,
  },
  {
    name: 'Completed',
    slug: 'completed',
    color: '#f06595',
    category: 'TASK_STATUS_CATEGORY_DONE',
    isDefault: false,
    isTerminal: true,
  },
];

const DEFAULT_PATHS: [string, string][] = [
  ['todo', 'on-progress'],
  ['on-progress', 'in-review'],
  ['in-review', 'completed'],
  ['in-review', 'on-progress'],
  ['completed', 'on-progress'],
];

export async function seedDefaultStatuses(profileId: string): Promise<void> {
  const created = new Map<string, string>();

  for (const [index, seed] of DEFAULT_STATUSES.entries()) {
    const status = await createStatus({
      profileId,
      name: seed.name,
      slug: seed.slug,
      color: seed.color,
      category: seed.category,
      isDefault: seed.isDefault,
      isTerminal: seed.isTerminal,
      position: index,
    });
    created.set(seed.slug, status.id);
  }

  for (const [from, to] of DEFAULT_PATHS) {
    const fromId = created.get(from);
    const toId = created.get(to);
    if (fromId && toId) {
      await createTransition({ profileId, fromStatusId: fromId, toStatusId: toId });
    }
  }
}
