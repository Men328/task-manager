import { Alert, Box, Button, Stack, Text, ThemeIcon } from '@mantine/core';
import { IconAlertTriangle, IconChecklist } from '@tabler/icons-react';
import { useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useSearchParams } from 'react-router-dom';

import { googleLoginUrl } from '../../api/identity';
import { useSession } from '../../context';
import { messageForCode } from '../../lib/errorCatalog';
import { tokens } from '../../theme';
import classes from './LoginPage.module.css';

function GoogleMark() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" aria-hidden="true" focusable="false">
      <path
        fill="#4285F4"
        d="M17.64 9.2c0-.64-.06-1.25-.16-1.84H9v3.48h4.84a4.14 4.14 0 0 1-1.8 2.72v2.26h2.92c1.7-1.57 2.68-3.88 2.68-6.62Z"
      />
      <path
        fill="#34A853"
        d="M9 18c2.43 0 4.47-.8 5.96-2.18l-2.92-2.26c-.8.54-1.84.86-3.04.86-2.34 0-4.32-1.58-5.03-3.7H.96v2.33A9 9 0 0 0 9 18Z"
      />
      <path
        fill="#FBBC05"
        d="M3.97 10.72a5.4 5.4 0 0 1 0-3.44V4.95H.96a9 9 0 0 0 0 8.1l3.01-2.33Z"
      />
      <path
        fill="#EA4335"
        d="M9 3.58c1.32 0 2.5.45 3.44 1.35l2.58-2.58C13.46.89 11.43 0 9 0A9 9 0 0 0 .96 4.95l3.01 2.33C4.68 5.16 6.66 3.58 9 3.58Z"
      />
    </svg>
  );
}

export function LoginPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { authenticated, signOut } = useSession();

  const errorCode = searchParams.get('error');
  const errorMessage = useMemo(
    () => (errorCode ? messageForCode(errorCode, i18n.resolvedLanguage ?? i18n.language) : null),
    [errorCode, i18n.resolvedLanguage, i18n.language],
  );

  useEffect(() => {
    if (authenticated) {
      navigate('/', { replace: true });
    }
  }, [authenticated, navigate]);

  return (
    <Box className={classes.root}>
      <Box className={classes.card}>
        <Stack gap="lg" align="center">
          <ThemeIcon size={54} radius={16} variant="gradient" gradient={{ from: tokens.brandFrom, to: tokens.brandTo, deg: 140 }}>
            <IconChecklist size={28} />
          </ThemeIcon>

          <Stack gap={6} align="center">
            <Text fz={22} fw={700} c={tokens.text} ta="center">
              {t('login.title')}
            </Text>
            <Text fz={13} c={tokens.textMuted} ta="center" maw={340}>
              {t('login.subtitle')}
            </Text>
          </Stack>

          {errorCode ? (
            <Alert
              color="red"
              variant="light"
              radius="md"
              icon={<IconAlertTriangle size={18} />}
              title={t('login.failedTitle')}
              w="100%"
            >
              <Text fz={12.5}>{errorMessage ?? t('login.failedFallback')}</Text>
            </Alert>
          ) : null}

          <Button
            size="md"
            radius="md"
            variant="default"
            fullWidth
            className={classes.googleButton}
            leftSection={<GoogleMark />}
            onClick={() => {
              signOut();
              window.location.assign(googleLoginUrl());
            }}
          >
            {t('login.withGoogle')}
          </Button>
        </Stack>
      </Box>
    </Box>
  );
}

export default LoginPage;
