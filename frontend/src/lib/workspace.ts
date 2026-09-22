const WORKSPACE_STORAGE_KEY = 'tm-workspace-id';

let cachedId: string | null = null;
let cacheLoaded = false;

function readStorage(): string | null {
  try {
    return window.localStorage.getItem(WORKSPACE_STORAGE_KEY);
  } catch {
    return null;
  }
}

export function readWorkspaceId(): string | null {
  if (!cacheLoaded) {
    cachedId = readStorage();
    cacheLoaded = true;
  }
  return cachedId;
}

export function writeWorkspaceId(id: string): void {
  cachedId = id;
  cacheLoaded = true;
  try {
    window.localStorage.setItem(WORKSPACE_STORAGE_KEY, id);
  } catch {}
}

export function clearWorkspaceId(): void {
  cachedId = null;
  cacheLoaded = true;
  try {
    window.localStorage.removeItem(WORKSPACE_STORAGE_KEY);
  } catch {}
}

export function slugifyWorkspaceName(value: string): string {
  return value
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/đ/g, 'd')
    .replace(/Đ/g, 'D')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

export function uniqueWorkspaceSlug(name: string): string {
  const base = slugifyWorkspaceName(name) || 'workspace';
  const suffix = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`.slice(-8);
  return `${base}-${suffix}`;
}
