import { Box, Loader, Stack, Text } from '@mantine/core';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { useAppDispatch } from '../../context';
import { restoreSession, setToken } from '../../context/session/sessionSlice';
import { tokens } from '../../theme';
import classes from './AuthCallbackPage.module.css';

export function AuthCallbackPage() {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const navigate = useNavigate();

  useEffect(() => {
    const fragment = new URLSearchParams(window.location.hash.replace(/^#/, ''));
    const token = fragment.get('token');
    const error = fragment.get('error');

    window.history.replaceState(null, '', window.location.pathname);

    if (error) {
      navigate(`/login?error=${encodeURIComponent(error)}`, { replace: true });
      return;
    }
    if (!token) {
      navigate('/login', { replace: true });
      return;
    }

    dispatch(setToken(token));
    void dispatch(restoreSession())
      .unwrap()
      .then(() => navigate('/', { replace: true }))
      .catch(() => navigate('/login?error=IDENTITY_AUTH_TOKEN_INVALID', { replace: true }));
  }, [dispatch, navigate]);

  return (
    <Box className={classes.root}>
      <Stack align="center" gap="sm">
        <Loader size="md" />
        <Text fz={13} c={tokens.textMuted}>
          {t('authCallback.signingIn')}
        </Text>
      </Stack>
    </Box>
  );
}

export default AuthCallbackPage;
