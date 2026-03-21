import { writable } from 'svelte/store';
import type { Check } from '$lib/types';
import { api } from '$lib/api';
import { toasts } from './toast';

interface ChecksState {
  checks: Check[];
  loading: boolean;
}

function createChecksStore() {
  const { subscribe, set, update } = writable<ChecksState>({
    checks: [],
    loading: true,
  });

  return {
    subscribe,
    async load() {
      update((s) => ({ ...s, loading: true }));
      try {
        const checks = await api.listChecks();
        set({ checks, loading: false });
      } catch (e) {
        toasts.error('Failed to load checks');
        update((s) => ({ ...s, loading: false }));
      }
    },
    updateCheck(updated: Check) {
      update((s) => ({
        ...s,
        checks: s.checks.map((c) => (c.id === updated.id ? updated : c)),
      }));
    },
    updateCheckStatus(checkId: string, status: Check['status']) {
      update((s) => ({
        ...s,
        checks: s.checks.map((c) =>
          c.id === checkId ? { ...c, status } : c
        ),
      }));
    },
    removeCheck(id: string) {
      update((s) => ({
        ...s,
        checks: s.checks.filter((c) => c.id !== id),
      }));
    },
    addCheck(check: Check) {
      update((s) => ({
        ...s,
        checks: [...s.checks, check].sort((a, b) => a.name.localeCompare(b.name)),
      }));
    },
  };
}

export const checksStore = createChecksStore();
