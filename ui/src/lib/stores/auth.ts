import { writable } from 'svelte/store';
import type { User } from '$lib/types';
import { api } from '$lib/api';

interface AuthState {
  user: User | null;
  loading: boolean;
  error: string | null;
}

function createAuthStore() {
  const { subscribe, set, update } = writable<AuthState>({
    user: null,
    loading: true,
    error: null,
  });

  return {
    subscribe,
    async load() {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const user = await api.me();
        set({ user, loading: false, error: null });
      } catch {
        set({ user: null, loading: false, error: null });
      }
    },
    logout() {
      set({ user: null, loading: false, error: null });
    },
  };
}

export const auth = createAuthStore();
