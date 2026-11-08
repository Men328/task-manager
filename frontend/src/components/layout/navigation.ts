import {
  IconArchive,
  IconArrowsExchange,
  IconCalendarEvent,
  IconCalendarMonth,
  IconChartHistogram,
  IconChecklist,
  IconFolders,
  IconMessages,
  IconSunHigh,
  IconUsers,
} from '@tabler/icons-react';

type IconComponent = typeof IconChecklist;

export interface NavItem {
  key: string;
  labelKey: string;
  icon: IconComponent;
  to: string;
  /** Khớp thêm các route con, vd. /messages/:peerId. */
  matchPrefix?: boolean;
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
    items: [
      { key: 'reports', labelKey: 'nav.reports', icon: IconChartHistogram, to: '/reports' },
    ],
  },
  {
    key: 'planning',
    labelKey: 'nav.groups.planning',
    items: [
      { key: 'today', labelKey: 'nav.today', icon: IconSunHigh, to: '/today' },
      { key: 'tasks', labelKey: 'nav.myTask', icon: IconChecklist, to: '/' },
      { key: 'calendar', labelKey: 'nav.calendar', icon: IconCalendarMonth, to: '/calendar' },
      { key: 'events', labelKey: 'nav.events', icon: IconCalendarEvent, to: '/events' },
      { key: 'backlog', labelKey: 'nav.backlog', icon: IconArchive, to: '/backlog' },
    ],
  },
  {
    key: 'social',
    labelKey: 'nav.groups.social',
    items: [
      { key: 'friends', labelKey: 'nav.friends', icon: IconUsers, to: '/friends' },
      { key: 'messages', labelKey: 'nav.messages', icon: IconMessages, to: '/messages', matchPrefix: true },
    ],
  },
  {
    key: 'config',
    labelKey: 'nav.groups.config',
    items: [
      { key: 'projects', labelKey: 'nav.projects', icon: IconFolders, to: '/projects' },
      {
        key: 'statuses',
        labelKey: 'nav.statusesLifecycle',
        icon: IconArrowsExchange,
        to: '/statuses',
      },
    ],
  },
];
