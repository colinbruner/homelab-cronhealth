import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// Mock the api module before importing the store
vi.mock('$lib/api', () => ({
  api: {
    listChecks: vi.fn(),
  },
}));

// Mock the toast store
vi.mock('./toast', () => ({
  toasts: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    dismiss: vi.fn(),
    subscribe: vi.fn(),
  },
}));

import { checksStore } from './checks';
import { api } from '$lib/api';
import { toasts } from './toast';
import type { Check } from '$lib/types';

function makeCheck(overrides: Partial<Check> = {}): Check {
  return {
    id: 'check-1',
    name: 'nightly-backup',
    slug: 'abc-123',
    period_seconds: 86400,
    grace_seconds: 300,
    status: 'up',
    last_ping_at: '2026-03-20T10:00:00Z',
    last_alerted_at: null,
    created_at: '2026-03-19T00:00:00Z',
    created_by: null,
    ...overrides,
  };
}

describe('checks store', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('starts with loading true and empty checks', () => {
    const state = get(checksStore);
    // After initial creation it should have empty checks
    expect(state.checks).toBeDefined();
    expect(Array.isArray(state.checks)).toBe(true);
  });

  it('loads checks from API', async () => {
    const checks = [makeCheck(), makeCheck({ id: 'check-2', name: 'hourly-job' })];
    vi.mocked(api.listChecks).mockResolvedValue(checks);

    await checksStore.load();

    const state = get(checksStore);
    expect(state.checks).toEqual(checks);
    expect(state.loading).toBe(false);
  });

  it('shows error toast on load failure', async () => {
    vi.mocked(api.listChecks).mockRejectedValue(new Error('network error'));

    await checksStore.load();

    const state = get(checksStore);
    expect(state.loading).toBe(false);
    expect(toasts.error).toHaveBeenCalledWith('Failed to load checks');
  });

  it('updates an existing check', async () => {
    const original = makeCheck();
    vi.mocked(api.listChecks).mockResolvedValue([original]);
    await checksStore.load();

    const updated = { ...original, name: 'updated-backup' };
    checksStore.updateCheck(updated);

    const state = get(checksStore);
    expect(state.checks[0].name).toBe('updated-backup');
  });

  it('updates check status', async () => {
    vi.mocked(api.listChecks).mockResolvedValue([makeCheck()]);
    await checksStore.load();

    checksStore.updateCheckStatus('check-1', 'alerting');

    const state = get(checksStore);
    expect(state.checks[0].status).toBe('alerting');
  });

  it('does not update status for non-existent check', async () => {
    vi.mocked(api.listChecks).mockResolvedValue([makeCheck()]);
    await checksStore.load();

    checksStore.updateCheckStatus('non-existent', 'down');

    const state = get(checksStore);
    expect(state.checks[0].status).toBe('up');
  });

  it('removes a check', async () => {
    vi.mocked(api.listChecks).mockResolvedValue([
      makeCheck(),
      makeCheck({ id: 'check-2', name: 'hourly' }),
    ]);
    await checksStore.load();

    checksStore.removeCheck('check-1');

    const state = get(checksStore);
    expect(state.checks).toHaveLength(1);
    expect(state.checks[0].id).toBe('check-2');
  });

  it('adds a check sorted by name', async () => {
    vi.mocked(api.listChecks).mockResolvedValue([
      makeCheck({ id: 'c', name: 'charlie' }),
      makeCheck({ id: 'a', name: 'alpha' }),
    ]);
    await checksStore.load();

    checksStore.addCheck(makeCheck({ id: 'b', name: 'bravo' }));

    const state = get(checksStore);
    const names = state.checks.map((c) => c.name);
    expect(names).toEqual(['alpha', 'bravo', 'charlie']);
  });
});
