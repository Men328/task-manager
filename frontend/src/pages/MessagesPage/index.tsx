import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import type { KeyboardEvent } from 'react';
import { ActionIcon, Avatar, Badge, Box, Button, Group, Text, Textarea, ThemeIcon, Tooltip, UnstyledButton } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconAlertCircle, IconArrowLeft, IconChecklist, IconMessages, IconSend } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';

import { getErrorMessage } from '../../api/client';
import { getMessage } from '../../api/message';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../../components/common/States';
import { useAppSelector, useChat, useFriends } from '../../context';
import { compareMessages, peerOf } from '../../lib/chatMessages';
import { matchSlashCommand } from '../../lib/slashCommands';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import type { ChatMessage, Friend, FriendProfile } from '../../types';
import { ChatTaskModal } from './ChatTaskModal';
import classes from './MessagesPage.module.css';
import { TaskCard } from './TaskCard';

const MAX_BODY_LENGTH = 2000;
/** Số trang cũ tối đa tải thêm khi tìm tin nhắn được notice trỏ tới. */
const MAX_FOCUS_PAGES = 20;
const FOCUS_HIGHLIGHT_MS = 2500;

function formatTime(value: string, language: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  const sameDay = date.toDateString() === new Date().toDateString();
  return sameDay
    ? date.toLocaleTimeString(language, { hour: '2-digit', minute: '2-digit' })
    : date.toLocaleString(language, { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' });
}

function ConversationRow({
  friend,
  last,
  unread,
  active,
  selfId,
  onSelect,
}: {
  friend: Friend;
  last: ChatMessage | undefined;
  unread: number;
  active: boolean;
  selfId: string | null;
  onSelect: () => void;
}) {
  const { t, i18n } = useTranslation();
  const text = last?.task ? `${t('chatTask.previewPrefix')} ${last.task.title}` : last?.body;
  const preview = last ? `${last.senderId === selfId ? `${t('chat.you')}: ` : ''}${text}` : t('chat.noMessagesYet');

  return (
    <UnstyledButton className={classes.conversation} data-active={active} onClick={onSelect}>
      <Group gap="sm" wrap="nowrap">
        <Avatar src={friend.profile.avatarUrl || null} name={friend.profile.displayName} color="initials" radius={999} size={38} />
        <Box miw={0} flex={1}>
          <Group justify="space-between" wrap="nowrap" gap={6}>
            <Text fz={13.5} fw={unread ? 800 : 700} c={tokens.text} truncate>
              {friend.profile.displayName}
            </Text>
            {last ? (
              <Text fz={11} c={tokens.textFaint} style={{ flexShrink: 0 }}>
                {formatTime(last.createdAt, i18n.language)}
              </Text>
            ) : null}
          </Group>
          <Group justify="space-between" wrap="nowrap" gap={6}>
            <Text fz={12} c={unread ? tokens.text : tokens.textMuted} fw={unread ? 600 : 400} truncate>
              {preview}
            </Text>
            {unread ? (
              <Badge size="sm" circle color="brand">
                {unread > 9 ? '9+' : unread}
              </Badge>
            ) : null}
          </Group>
        </Box>
      </Group>
    </UnstyledButton>
  );
}

function MessageBubble({
  message,
  mine,
  peerName,
  focused,
  onRetry,
}: {
  message: ChatMessage;
  mine: boolean;
  peerName: string;
  focused: boolean;
  onRetry: () => void;
}) {
  const { t, i18n } = useTranslation();

  return (
    <Box className={classes.bubbleRow} data-mine={mine} data-focused={focused} data-message-id={message.id}>
      {message.task ? (
        <TaskCard task={message.task} mine={mine} peerName={peerName} />
      ) : (
        <Box className={classes.bubble} data-mine={mine} data-pending={Boolean(message.pending)}>
          <Text fz={13.5} className={classes.bubbleText}>
            {message.body}
          </Text>
        </Box>
      )}
      <Group gap={4} className={classes.meta} data-mine={mine}>
        {message.failed ? (
          <>
            <IconAlertCircle size={12} color="var(--mantine-color-red-6)" />
            <Text fz={11} c="red.6">
              {t('chat.sendFailed')}
            </Text>
            <Button size="compact-xs" variant="subtle" color="red" onClick={onRetry}>
              {t('chat.retry')}
            </Button>
          </>
        ) : (
          <Text fz={11} c={tokens.textFaint}>
            {message.pending ? t('chat.sending') : formatTime(message.createdAt, i18n.language)}
          </Text>
        )}
      </Group>
    </Box>
  );
}

function Thread({ peer, onBack }: { peer: FriendProfile; onBack: () => void }) {
  const { t } = useTranslation();
  const { profileId, thread, loadLatest, loadOlder, send, sendTask, activate } = useChat(peer.id);
  const [params, setParams] = useSearchParams();
  const focusId = params.get('focus');
  const [draft, setDraft] = useState('');
  const [slashDismissed, setSlashDismissed] = useState(false);
  const [taskTitle, setTaskTitle] = useState<string | null>(null);
  const [highlightId, setHighlightId] = useState<string | null>(null);
  const focusPages = useRef(0);
  const scrollRef = useRef<HTMLDivElement>(null);
  const stickToBottom = useRef(true);
  const lastId = thread.messages[thread.messages.length - 1]?.id;
  const slash = slashDismissed ? null : matchSlashCommand(draft);

  useEffect(() => {
    focusPages.current = 0;
  }, [focusId]);

  useEffect(() => {
    if (!focusId || !thread.loaded || thread.loading) {
      return;
    }
    const clearFocus = () => {
      setParams(
        (current) => {
          const next = new URLSearchParams(current);
          next.delete('focus');
          return next;
        },
        { replace: true },
      );
    };
    const found = thread.messages.some((message) => message.id === focusId);
    if (found) {
      stickToBottom.current = false;
      const element = scrollRef.current?.querySelector(`[data-message-id="${CSS.escape(focusId)}"]`);
      element?.scrollIntoView({ block: 'center' });
      setHighlightId(focusId);
      clearFocus();
      return;
    }
    if (thread.hasMore && focusPages.current < MAX_FOCUS_PAGES) {
      focusPages.current += 1;
      loadOlder();
      return;
    }
    clearFocus();
  }, [focusId, thread.loaded, thread.loading, thread.messages, thread.hasMore, loadOlder, setParams]);

  useEffect(() => {
    if (!highlightId) {
      return;
    }
    const timer = window.setTimeout(() => setHighlightId(null), FOCUS_HIGHLIGHT_MS);
    return () => window.clearTimeout(timer);
  }, [highlightId]);

  useEffect(() => {
    activate(peer.id);
    loadLatest();
    stickToBottom.current = !focusId;
    return () => activate(null);
  }, [activate, loadLatest, peer.id]);

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (element && stickToBottom.current) {
      element.scrollTop = element.scrollHeight;
    }
  }, [lastId, thread.loaded]);

  const onScroll = () => {
    const element = scrollRef.current;
    if (element) {
      stickToBottom.current = element.scrollHeight - element.scrollTop - element.clientHeight < 80;
    }
  };

  const openTaskModal = (title: string) => {
    setTaskTitle(title);
    setDraft('');
  };

  const changeDraft = (value: string) => {
    if (!value.startsWith('/')) {
      setSlashDismissed(false);
    }
    setDraft(value);
  };

  const submit = () => {
    if (slash) {
      openTaskModal(slash.argument);
      return;
    }
    const body = draft.trim();
    if (!body || body.length > MAX_BODY_LENGTH) {
      return;
    }
    stickToBottom.current = true;
    send(body);
    setDraft('');
  };

  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Escape' && slash) {
      event.preventDefault();
      setSlashDismissed(true);
      return;
    }
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault();
      submit();
    }
  };

  return (
    <Box className={classes.thread}>
      <Group className={classes.threadHeader} gap="sm" wrap="nowrap">
        <ActionIcon variant="subtle" color="gray" className={classes.backButton} onClick={onBack} aria-label={t('chat.back')}>
          <IconArrowLeft size={18} />
        </ActionIcon>
        <Avatar src={peer.avatarUrl || null} name={peer.displayName} color="initials" radius={999} size={34} />
        <Box miw={0}>
          <Text fz={14} fw={700} c={tokens.text} truncate>
            {peer.displayName}
          </Text>
          <Text fz={12} c={tokens.textMuted} truncate>
            {peer.email}
          </Text>
        </Box>
      </Group>

      <Box ref={scrollRef} onScroll={onScroll} className={`${scrollClasses.scroll} ${classes.messages}`}>
        {thread.hasMore ? (
          <Group justify="center" mb="sm">
            <Button size="compact-xs" variant="default" loading={thread.loading} onClick={loadOlder}>
              {t('chat.loadOlder')}
            </Button>
          </Group>
        ) : null}
        {thread.error ? <ApiErrorAlert message={thread.error} onRetry={loadLatest} /> : null}
        {!thread.loaded && thread.loading && thread.messages.length === 0 ? <LoadingBlock /> : null}
        {thread.loaded && thread.messages.length === 0 ? (
          <CenteredPanel icon={IconMessages} title={t('chat.emptyThreadTitle')} description={t('chat.emptyThread', { name: peer.displayName })} />
        ) : null}
        {thread.messages.map((message) => (
          <MessageBubble
            key={message.clientId || message.id}
            message={message}
            mine={message.pending || message.failed ? true : message.senderId === profileId}
            peerName={peer.displayName}
            focused={highlightId === message.id}
            onRetry={() => send(message.body, message.clientId)}
          />
        ))}
      </Box>

      <Box className={classes.composerWrap}>
        {slash ? (
          <Box className={classes.slashMenu} role="listbox" aria-label={t('chatTask.menuLabel')}>
            {slash.commands.map((command, index) => (
              <UnstyledButton
                key={command.id}
                className={classes.slashItem}
                data-active={index === 0}
                role="option"
                aria-selected={index === 0}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => openTaskModal(slash.argument)}
              >
                <Group gap="sm" wrap="nowrap">
                  <ThemeIcon size={28} radius={8} variant="light" color="brand">
                    <IconChecklist size={16} />
                  </ThemeIcon>
                  <Box miw={0}>
                    <Text fz={13} fw={700} c={tokens.text}>
                      /task · {t('chatTask.command', { name: peer.displayName })}
                    </Text>
                    <Text fz={11.5} c={tokens.textMuted} truncate>
                      {slash.argument ? t('chatTask.commandWithTitle', { title: slash.argument }) : t('chatTask.commandHint')}
                    </Text>
                  </Box>
                </Group>
              </UnstyledButton>
            ))}
          </Box>
        ) : null}
        <Group className={classes.composer} gap="sm" wrap="nowrap" align="flex-end">
          <Tooltip label={t('chatTask.command', { name: peer.displayName })}>
            <ActionIcon size={36} radius="md" variant="default" onClick={() => openTaskModal('')} aria-label={t('chatTask.command', { name: peer.displayName })}>
              <IconChecklist size={17} />
            </ActionIcon>
          </Tooltip>
          <Textarea
            flex={1}
            autosize
            minRows={1}
            maxRows={5}
            placeholder={t('chat.placeholder')}
            aria-label={t('chat.placeholder')}
            value={draft}
            maxLength={MAX_BODY_LENGTH}
            onChange={(event) => changeDraft(event.currentTarget.value)}
            onKeyDown={onKeyDown}
          />
          <Tooltip label={t('chat.send')}>
            <ActionIcon size={36} radius="md" disabled={!draft.trim()} onClick={submit} aria-label={t('chat.send')}>
              <IconSend size={17} />
            </ActionIcon>
          </Tooltip>
        </Group>
      </Box>

      <ChatTaskModal
        opened={taskTitle !== null}
        peerName={peer.displayName}
        initialTitle={taskTitle ?? ''}
        onClose={() => setTaskTitle(null)}
        onSubmit={(value) => {
          stickToBottom.current = true;
          return sendTask(value);
        }}
      />
    </Box>
  );
}

export function MessagesPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { peerId } = useParams<{ peerId: string }>();
  const [params] = useSearchParams();
  const focusId = params.get('focus');
  const token = useAppSelector((state) => state.session.token);
  const { friends, loading, error, reload } = useFriends();
  const { profileId, lastByPeer, unreadByPeer, conversationsError, reloadConversations } = useChat(null);

  useEffect(() => {
    if (peerId || !focusId || !token || !profileId) {
      return;
    }
    let cancelled = false;
    getMessage(token, focusId)
      .then((message) => {
        if (!cancelled) {
          navigate(`/messages/${peerOf(message, profileId)}?focus=${encodeURIComponent(message.id)}`, { replace: true });
        }
      })
      .catch((cause: unknown) => {
        if (!cancelled) {
          notifications.show({ title: t('chatTask.openFailed'), message: getErrorMessage(cause), color: 'red' });
          navigate('/messages', { replace: true });
        }
      });
    return () => {
      cancelled = true;
    };
  }, [peerId, focusId, token, profileId, navigate, t]);

  useEffect(() => {
    reload();
  }, [reload]);

  const sorted = useMemo(
    () =>
      [...friends].sort((a, b) => {
        const left = lastByPeer[a.profile.id];
        const right = lastByPeer[b.profile.id];
        if (left && right) {
          return compareMessages(right, left);
        }
        if (left || right) {
          return left ? -1 : 1;
        }
        return a.profile.displayName.localeCompare(b.profile.displayName);
      }),
    [friends, lastByPeer],
  );

  const peer = friends.find((friend) => friend.profile.id === peerId)?.profile ?? null;

  return (
    <Box className={classes.root} data-has-peer={Boolean(peerId)}>
      <Box className={classes.sidebar}>
        <Group className={classes.sidebarHeader} gap={9} wrap="nowrap">
          <ThemeIcon size={30} radius={9} variant="light" color="brand">
            <IconMessages size={17} />
          </ThemeIcon>
          <div>
            <Text fz={17} fw={800} c={tokens.text}>
              {t('chat.pageTitle')}
            </Text>
            <Text fz={12} c={tokens.textMuted}>
              {t('chat.pageSubtitle')}
            </Text>
          </div>
        </Group>
        <Box className={`${scrollClasses.scroll} ${classes.conversationList}`}>
          {error ? <ApiErrorAlert message={error} onRetry={reload} /> : null}
          {conversationsError ? <ApiErrorAlert message={conversationsError} onRetry={reloadConversations} /> : null}
          {!error && loading && friends.length === 0 ? <LoadingBlock /> : null}
          {!error && !loading && friends.length === 0 ? (
            <CenteredPanel icon={IconMessages} title={t('chat.noFriendsTitle')} description={t('chat.noFriends')} />
          ) : null}
          {sorted.map((friend) => (
            <ConversationRow
              key={friend.profile.id}
              friend={friend}
              last={lastByPeer[friend.profile.id]}
              unread={unreadByPeer[friend.profile.id] ?? 0}
              active={friend.profile.id === peerId}
              selfId={profileId}
              onSelect={() => navigate(`/messages/${friend.profile.id}`)}
            />
          ))}
        </Box>
      </Box>

      <Box className={classes.main}>
        {peer ? (
          <Thread key={peer.id} peer={peer} onBack={() => navigate('/messages')} />
        ) : (
          <CenteredPanel
            icon={IconMessages}
            title={peerId && !loading ? t('chat.notFriendTitle') : t('chat.pickTitle')}
            description={peerId && !loading ? t('chat.notFriend') : t('chat.pick')}
          />
        )}
      </Box>
    </Box>
  );
}

export default MessagesPage;
