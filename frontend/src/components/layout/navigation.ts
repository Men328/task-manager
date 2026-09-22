import { IconArrowsExchange, IconChecklist } from '@tabler/icons-react';

type IconComponent = typeof IconChecklist;

export interface NavItem {
  key: string;
  labelKey: string;
  icon: IconComponent;
  to: string;
}

export interface NavSection {
  key: string;
  labelKey: string;
  items: NavItem[];
}

export const NAV_SECTIONS: NavSection[] = [
  {
    key: 'dashboard',
    labelKey: 'nav.groups.dashboard',
    items: [],
  },
  {
    key: 'planning',
    labelKey: 'nav.groups.planning',
    items: [{ key: 'tasks', labelKey: 'nav.myTask', icon: IconChecklist, to: '/' }],
  },
  {
    key: 'config',
    labelKey: 'nav.groups.config',
    items: [
      {
        key: 'statuses',
        labelKey: 'nav.statusesLifecycle',
        icon: IconArrowsExchange,
        to: '/statuses',
      },
    ],
  },
];
