const TOKEN_STORAGE_KEY = 'tm-session-token';

let cachedToken: string | null = null;
let cacheLoaded = false;

function readStorage(): string | null {
  try {
    return window.localStorage.getItem(TOKEN_STORAGE_KEY);
  } catch {
    return null;
  }
}

export function readToken(): string | null {
  if (!cacheLoaded) {
    cachedToken = readStorage();
    cacheLoaded = true;
  }
  return cachedToken;
}

export function writeToken(token: string): void {
  cachedToken = token;
  cacheLoaded = true;
  try {
    window.localStorage.setItem(TOKEN_STORAGE_KEY, token);
  } catch {}
}

export function clearToken(): void {
  cachedToken = null;
  cacheLoaded = true;
  try {
    window.localStorage.removeItem(TOKEN_STORAGE_KEY);
  } catch {}
}
