import { useEffect, useState } from 'react';
import type { FormEvent, ReactNode } from 'react';
import {
  Avatar,
  Badge,
  Box,
  Button,
  Flex,
  Group,
  Modal,
  SegmentedControl,
  Text,
  TextInput,
  ThemeIcon,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconMessage, IconSearch, IconUsers } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { getErrorMessage } from '../../api/client';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../../components/common/States';
import { useFriends } from '../../context';
import { relationAction } from '../../lib/friendRelation';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import type { FriendProfile, FriendSearchResult } from '../../types';
import classes from './FriendsPage.module.css';

type Tab = 'friends' | 'incoming' | 'outgoing';

function PersonRow({ profile, actions }: { profile: FriendProfile; actions: ReactNode }) {
  return (
    <Group className={classes.row} justify="space-between" wrap="wrap" gap="sm">
      <Group gap="sm" wrap="nowrap" className={classes.identity}>
        <Avatar src={profile.avatarUrl || null} name={profile.displayName} color="initials" radius={999} size={36} />
        <Box miw={0}>
          <Text fz={13.5} fw={700} c={tokens.text} truncate>
            {profile.displayName}
          </Text>
          <Text fz={12} c={tokens.textMuted} truncate>
            {profile.email}
          </Text>
        </Box>
      </Group>
      <Group gap="xs">{actions}</Group>
    </Group>
  );
}

export function FriendsPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { friends, incoming, outgoing, loading, error, reload, search, send, accept, decline, cancel, remove } =
    useFriends();
  const [tab, setTab] = useState<Tab>('friends');
  const [email, setEmail] = useState('');
  const [searchedEmail, setSearchedEmail] = useState('');
  const [result, setResult] = useState<FriendSearchResult | null>(null);
  const [searching, setSearching] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [removing, setRemoving] = useState<FriendProfile | null>(null);

  useEffect(() => {
    reload();
  }, [reload]);

  const runSearch = async (value: string) => {
    setSearching(true);
    try {
      setResult(await search(value));
      setSearchedEmail(value);
    } catch (cause) {
      setResult(null);
      notifications.show({ title: t('friends.searchFailed'), message: getErrorMessage(cause), color: 'red' });
    } finally {
      setSearching(false);
    }
  };

  const submitSearch = (event: FormEvent) => {
    event.preventDefault();
    const value = email.trim();
    if (value) {
      void runSearch(value);
    }
  };

  /** Chặn bấm đúp qua `busyId`; xong thì làm mới kết quả tìm kiếm đang hiện. */
  const act = async (id: string, action: () => Promise<void>) => {
    setBusyId(id);
    try {
      await action();
      if (result?.profile) {
        await runSearch(searchedEmail);
      }
    } catch (cause) {
      notifications.show({ title: t('friends.actionFailed'), message: getErrorMessage(cause), color: 'red' });
      reload();
    } finally {
      setBusyId(null);
    }
  };

  const confirmRemove = async () => {
    if (removing) {
      await act(removing.id, () => remove(removing.id));
      setRemoving(null);
    }
  };

  const searchAction = (profile: FriendProfile, relation: FriendSearchResult['relation']) => {
    switch (relationAction(relation)) {
      case 'add':
        return (
          <Button size="xs" loading={busyId === profile.id} onClick={() => void act(profile.id, () => send(profile.id))}>
            {t('friends.addAction')}
          </Button>
        );
      case 'accept':
        // Gửi lời mời khi đối phương đã mời mình sẽ tự chấp nhận, nên không cần id lời mời.
        return (
          <Button size="xs" loading={busyId === profile.id} onClick={() => void act(profile.id, () => send(profile.id))}>
            {t('friends.acceptAction')}
          </Button>
        );
      case 'sent':
        return (
          <Badge variant="light" color="gray">
            {t('friends.statusSent')}
          </Badge>
        );
      case 'friend':
        return (
          <Badge variant="light" color="brand">
            {t('friends.statusFriend')}
          </Badge>
        );
      default:
        return null;
    }
  };

  const hasData = friends.length + incoming.length + outgoing.length > 0;
  const isEmpty =
    (tab === 'friends' && friends.length === 0) ||
    (tab === 'incoming' && incoming.length === 0) ||
    (tab === 'outgoing' && outgoing.length === 0);
  const emptyKey = { friends: 'Friends', incoming: 'Incoming', outgoing: 'Outgoing' }[tab];

  return (
    <Flex direction="column" h="100%" className={classes.root}>
      <Box px="lg" pt="md" className={classes.headerBar}>
        <Group justify="space-between" align="center" mb="md" wrap="wrap" gap="sm">
          <Group gap={9} wrap="nowrap">
            <ThemeIcon size={30} radius={9} variant="light" color="brand">
              <IconUsers size={17} />
            </ThemeIcon>
            <div>
              <Text fz={19} fw={800} c={tokens.text}>
                {t('friends.pageTitle')}
              </Text>
              <Text fz={12} c={tokens.textMuted}>
                {t('friends.pageSubtitle')}
              </Text>
            </div>
          </Group>
          <SegmentedControl
            value={tab}
            onChange={(value) => setTab(value as Tab)}
            data={[
              { value: 'friends', label: t('friends.tabFriends') },
              { value: 'incoming', label: `${t('friends.tabIncoming')} (${incoming.length})` },
              { value: 'outgoing', label: t('friends.tabOutgoing') },
            ]}
          />
        </Group>
      </Box>

      <Box className={`${scrollClasses.scroll} ${classes.scrollArea}`} p="lg">
        <form onSubmit={submitSearch}>
          <Group gap="sm" mb="md" wrap="nowrap">
            <TextInput
              flex={1}
              type="email"
              aria-label={t('friends.searchPlaceholder')}
              placeholder={t('friends.searchPlaceholder')}
              leftSection={<IconSearch size={15} />}
              value={email}
              onChange={(event) => setEmail(event.currentTarget.value)}
            />
            <Button type="submit" loading={searching} disabled={!email.trim()}>
              {t('friends.searchAction')}
            </Button>
          </Group>
        </form>

        {result ? (
          <Box className={classes.panel} mb="md">
            {result.profile ? (
              <PersonRow profile={result.profile} actions={searchAction(result.profile, result.relation)} />
            ) : (
              <Text className={classes.row} fz={13} c={tokens.textMuted}>
                {t('friends.searchEmpty')}
              </Text>
            )}
          </Box>
        ) : null}

        {error ? <ApiErrorAlert message={error} onRetry={reload} /> : null}
        {!error && loading && !hasData ? <LoadingBlock /> : null}

        {!error && !(loading && !hasData) && isEmpty ? (
          <CenteredPanel
            icon={IconUsers}
            title={t(`friends.empty${emptyKey}Title`)}
            description={t(`friends.empty${emptyKey}`)}
          />
        ) : null}

        {!error && !isEmpty ? (
          <Box className={classes.panel}>
            {tab === 'friends'
              ? friends.map((friend) => (
                  <PersonRow
                    key={friend.profile.id}
                    profile={friend.profile}
                    actions={
                      <>
                        <Button
                          size="xs"
                          variant="light"
                          leftSection={<IconMessage size={14} />}
                          onClick={() => navigate(`/messages/${friend.profile.id}`)}
                        >
                          {t('friends.messageAction')}
                        </Button>
                        <Button size="xs" variant="default" onClick={() => setRemoving(friend.profile)}>
                          {t('friends.removeAction')}
                        </Button>
                      </>
                    }
                  />
                ))
              : null}
            {tab === 'incoming'
              ? incoming.map((request) => (
                  <PersonRow
                    key={request.id}
                    profile={request.profile}
                    actions={
                      <>
                        <Button
                          size="xs"
                          loading={busyId === request.id}
                          onClick={() => void act(request.id, () => accept(request.id))}
                        >
                          {t('friends.acceptAction')}
                        </Button>
                        <Button
                          size="xs"
                          variant="default"
                          disabled={busyId === request.id}
                          onClick={() => void act(request.id, () => decline(request.id))}
                        >
                          {t('friends.declineAction')}
                        </Button>
                      </>
                    }
                  />
                ))
              : null}
            {tab === 'outgoing'
              ? outgoing.map((request) => (
                  <PersonRow
                    key={request.id}
                    profile={request.profile}
                    actions={
                      <Button
                        size="xs"
                        variant="default"
                        loading={busyId === request.id}
                        onClick={() => void act(request.id, () => cancel(request.id))}
                      >
                        {t('friends.cancelAction')}
                      </Button>
                    }
                  />
                ))
              : null}
          </Box>
        ) : null}
      </Box>

      <Modal opened={removing !== null} onClose={() => setRemoving(null)} title={t('friends.removeTitle')} centered>
        <Text fz={13.5} mb="md">
          {t('friends.removeMessage', { name: removing?.displayName ?? '' })}
        </Text>
        <Group justify="flex-end" gap="sm">
          <Button variant="default" onClick={() => setRemoving(null)}>
            {t('friends.removeCancel')}
          </Button>
          <Button color="red" loading={removing !== null && busyId === removing.id} onClick={() => void confirmRemove()}>
            {t('friends.removeAction')}
          </Button>
        </Group>
      </Modal>
    </Flex>
  );
}
