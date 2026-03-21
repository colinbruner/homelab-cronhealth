# SvelteKit Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the cronhealth SvelteKit SPA frontend with dashboard, check detail, new/edit check, alerts feed, and settings screens — all backed by real-time SSE updates and OIDC auth.

**Architecture:** SvelteKit in SPA mode (static adapter) served by nginx. All API calls go through a typed fetch wrapper (`lib/api.ts`). Real-time updates via EventSource connecting to `/api/events`. Auth state tracked in a Svelte store that redirects to `/auth/login` on 401. Tailwind CSS with design tokens from DESIGN.md (dark theme, Inter + JetBrains Mono fonts).

**Tech Stack:** SvelteKit 2, Svelte 5, TypeScript, Tailwind CSS 4, static adapter, vitest + @testing-library/svelte

**Reference:** All API endpoints, design tokens, screen layouts, and interaction states are defined in `DESIGN.md` (lines 200-850).

---

## File Structure

```
ui/
├── package.json
├── svelte.config.js          # SvelteKit config with static adapter
├── vite.config.ts            # Vite config with vitest
├── tsconfig.json
├── tailwind.config.js        # Design tokens (colors, fonts, spacing)
├── src/
│   ├── app.html              # Shell HTML (fonts, meta)
│   ├── app.css               # Tailwind directives + global styles
│   ├── lib/
│   │   ├── types.ts          # TypeScript types matching Go models
│   │   ├── api.ts            # Typed fetch wrapper for all /api/* endpoints
│   │   ├── sse.ts            # EventSource wrapper with reconnect
│   │   ├── stores/
│   │   │   ├── auth.ts       # Auth state store (user info, 401 redirect)
│   │   │   └── checks.ts     # Checks list store with SSE live updates
│   │   └── components/
│   │       ├── StatusBadge.svelte    # Status text + color badge
│   │       ├── CheckCard.svelte      # Dashboard check card
│   │       ├── SkeletonCard.svelte   # Loading placeholder card
│   │       ├── SkeletonRow.svelte    # Loading placeholder table row
│   │       ├── Toast.svelte          # Toast notification
│   │       ├── ToastContainer.svelte # Toast stack manager
│   │       ├── Modal.svelte          # Reusable modal dialog
│   │       ├── TimeAgo.svelte        # Human-readable time delta
│   │       ├── Nav.svelte            # Top nav bar (desktop) / bottom bar (mobile)
│   │       └── EmptyState.svelte     # Reusable empty state with CTA
│   └── routes/
│       ├── +layout.svelte    # Root layout: nav, toast container, auth guard
│       ├── +layout.ts        # Load auth state on every navigation
│       ├── +page.svelte      # Dashboard (/)
│       ├── checks/
│       │   ├── new/
│       │   │   └── +page.svelte    # New check form
│       │   └── [id]/
│       │       ├── +page.svelte    # Check detail
│       │       └── edit/
│       │           └── +page.svelte # Edit check form
│       ├── alerts/
│       │   └── +page.svelte        # Alerts feed
│       └── settings/
│           └── +page.svelte        # Settings / profile
└── static/
    └── favicon.svg
```

---

## Task 1: SvelteKit Project Scaffolding

**Files:**
- Create: `ui/package.json`
- Create: `ui/svelte.config.js`
- Create: `ui/vite.config.ts`
- Create: `ui/tsconfig.json`
- Create: `ui/tailwind.config.js`
- Create: `ui/src/app.html`
- Create: `ui/src/app.css`

- [ ] **Step 1: Scaffold SvelteKit project**

```bash
cd /Users/colinbruner/code/colinbruner/homelab-cronhealth
npx sv create ui --template minimal --types ts --no-add-ons --no-install
```

- [ ] **Step 2: Install dependencies**

```bash
cd ui
npm install
npm install -D @sveltejs/adapter-static tailwindcss @tailwindcss/vite
npm install -D vitest @testing-library/svelte jsdom
```

- [ ] **Step 3: Configure static adapter in `ui/svelte.config.js`**

```js
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      fallback: 'index.html'  // SPA mode: all routes serve index.html
    })
  }
};
```

- [ ] **Step 4: Configure Tailwind in `ui/vite.config.ts`**

```ts
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  test: {
    include: ['src/**/*.test.ts'],
    environment: 'jsdom',
    setupFiles: []
  }
});
```

- [ ] **Step 5: Configure Tailwind design tokens in `ui/tailwind.config.js`**

```js
/** @type {import('tailwindcss').Config} */
export default {
  // Note: Tailwind v4 handles content detection automatically.
  // The content array is not needed but kept for compatibility with tooling.
  theme: {
    extend: {
      colors: {
        bg: '#0f1117',
        surface: '#1a1d24',
        border: '#2a2d36',
        'text-primary': '#e2e8f0',
        'text-secondary': '#94a3b8',
        status: {
          up: '#22c55e',
          down: '#ef4444',
          silenced: '#6b7280',
          new: '#f59e0b',
          alerting: '#f97316',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'monospace'],
      },
      fontSize: {
        body: '14px',
      },
      borderRadius: {
        card: '4px',
        badge: '2px',
      },
    },
  },
};
```

- [ ] **Step 6: Set up `ui/src/app.css`**

```css
@import 'tailwindcss';
@config '../tailwind.config.js';

@layer base {
  body {
    @apply bg-bg text-text-primary text-body font-sans antialiased;
  }
}
```

- [ ] **Step 7: Set up `ui/src/app.html`**

```html
<!doctype html>
<html lang="en" class="dark">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet" />
    <title>cronhealth</title>
    %sveltekit.head%
  </head>
  <body data-sveltekit-preload-data="hover">
    <div style="display: contents">%sveltekit.body%</div>
  </body>
</html>
```

- [ ] **Step 8: Create SPA fallback layout**

Create `ui/src/routes/+layout.ts`:
```ts
export const prerender = false;
export const ssr = false;
```

Create minimal `ui/src/routes/+layout.svelte`:
```svelte
<script>
  import '../app.css';
  let { children } = $props();
</script>

{@render children()}
```

Create minimal `ui/src/routes/+page.svelte`:
```svelte
<h1 class="text-text-primary text-2xl p-8">cronhealth</h1>
```

- [ ] **Step 9: Verify dev server starts**

```bash
cd ui && npm run dev -- --port 5173
```

Visit http://localhost:5173 — should show "cronhealth" on dark background with Inter font.

- [ ] **Step 10: Verify build works**

```bash
cd ui && npm run build
```

Should produce `ui/build/` directory with `index.html` as fallback.

- [ ] **Step 11: Commit**

```bash
git add ui/
git commit -m "feat(ui): scaffold SvelteKit project with static adapter and Tailwind"
```

---

## Task 2: TypeScript Types and API Client

**Files:**
- Create: `ui/src/lib/types.ts`
- Create: `ui/src/lib/api.ts`

- [ ] **Step 1: Create TypeScript types matching Go models in `ui/src/lib/types.ts`**

These must match the JSON output of the Go handlers in `internal/db/models.go` and `internal/api/handlers.go`.

```ts
export type CheckStatus = 'new' | 'up' | 'down' | 'alerting' | 'silenced';

export interface Check {
  id: string;
  name: string;
  slug: string;
  period_seconds: number;
  grace_seconds: number;
  status: CheckStatus;
  last_ping_at: string | null;
  last_alerted_at: string | null;
  created_at: string;
  created_by: string | null;
}

export interface Ping {
  id: string;
  check_id: string;
  received_at: string;
  source_ip: string | null;
  exit_code: number | null;
}

export interface Alert {
  id: string;
  check_id: string;
  started_at: string;
  resolved_at: string | null;
  alert_count: number;
  check_name?: string;
}

export interface Silence {
  id: string;
  check_id: string;
  silenced_by: string | null;
  starts_at: string;
  ends_at: string | null;
  reason: string | null;
  created_at: string;
}

export interface User {
  user_id: string;
  email: string;
}

export interface NotificationChannel {
  id: string;
  user_id: string;
  label: string;
  type: 'email' | 'sms';
  target: string;
  enabled: boolean;
  created_at: string;
}

export interface CreateCheckRequest {
  name: string;
  period_seconds: number;
  grace_seconds?: number;
  channel_ids?: string[];
}

export interface UpdateCheckRequest {
  name: string;
  period_seconds: number;
  grace_seconds?: number;
}

export interface SnoozeRequest {
  duration_minutes: number;
}

export interface SilenceRequest {
  reason?: string;
  ends_at?: string;
}

export interface ApiError {
  error: string;
}
```

- [ ] **Step 2: Create API client in `ui/src/lib/api.ts`**

This wraps fetch with JSON handling, auth error detection, and typed responses. On 401, it redirects to `/auth/login`.

```ts
import type {
  Check, Ping, Alert, Silence, User, NotificationChannel,
  CreateCheckRequest, UpdateCheckRequest, SnoozeRequest, SilenceRequest
} from './types';

class ApiClient {
  private async request<T>(path: string, options?: RequestInit): Promise<T> {
    const res = await fetch(path, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });

    if (res.status === 401) {
      window.location.href = '/auth/login';
      throw new Error('Authentication required');
    }

    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }));
      throw new Error(body.error || res.statusText);
    }

    return res.json();
  }

  // Checks
  listChecks(): Promise<Check[]> {
    return this.request('/api/checks');
  }

  getCheck(id: string): Promise<Check> {
    return this.request(`/api/checks/${id}`);
  }

  createCheck(data: CreateCheckRequest): Promise<Check> {
    return this.request('/api/checks', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  updateCheck(id: string, data: UpdateCheckRequest): Promise<Check> {
    return this.request(`/api/checks/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  deleteCheck(id: string): Promise<{ ok: boolean }> {
    return this.request(`/api/checks/${id}`, { method: 'DELETE' });
  }

  // Pings
  listPings(checkId: string, limit = 50, offset = 0): Promise<Ping[]> {
    return this.request(`/api/checks/${checkId}/pings?limit=${limit}&offset=${offset}`);
  }

  // Snooze / Silence
  snoozeCheck(id: string, data: SnoozeRequest): Promise<Silence> {
    return this.request(`/api/checks/${id}/snooze`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  silenceCheck(id: string, data: SilenceRequest): Promise<Silence> {
    return this.request(`/api/checks/${id}/silence`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  removeSilence(id: string): Promise<{ ok: boolean }> {
    return this.request(`/api/checks/${id}/silence`, { method: 'DELETE' });
  }

  // Alerts
  listAlerts(): Promise<Alert[]> {
    return this.request('/api/alerts');
  }

  getAlert(id: string): Promise<Alert> {
    return this.request(`/api/alerts/${id}`);
  }

  // Auth
  me(): Promise<User> {
    return this.request('/api/me');
  }
}

export const api = new ApiClient();
```

- [ ] **Step 3: Commit**

```bash
git add ui/src/lib/types.ts ui/src/lib/api.ts
git commit -m "feat(ui): add TypeScript types and typed API client"
```

---

## Task 3: Stores (Auth + Checks + Toast)

**Files:**
- Create: `ui/src/lib/stores/auth.ts`
- Create: `ui/src/lib/stores/checks.ts`
- Create: `ui/src/lib/stores/toast.ts`

- [ ] **Step 1: Create auth store in `ui/src/lib/stores/auth.ts`**

```ts
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
        // 401 is handled by api.ts redirect — no error shown here
      }
    },
    logout() {
      set({ user: null, loading: false, error: null });
    },
  };
}

export const auth = createAuthStore();
```

- [ ] **Step 2: Create toast store in `ui/src/lib/stores/toast.ts`**

```ts
import { writable } from 'svelte/store';

export interface Toast {
  id: string;
  message: string;
  type: 'success' | 'error' | 'info';
}

function createToastStore() {
  const { subscribe, update } = writable<Toast[]>([]);

  function add(message: string, type: Toast['type'] = 'info') {
    const id = crypto.randomUUID();
    update((toasts) => [...toasts, { id, message, type }]);
    setTimeout(() => dismiss(id), 4000);
  }

  function dismiss(id: string) {
    update((toasts) => toasts.filter((t) => t.id !== id));
  }

  return {
    subscribe,
    success: (msg: string) => add(msg, 'success'),
    error: (msg: string) => add(msg, 'error'),
    info: (msg: string) => add(msg, 'info'),
    dismiss,
  };
}

export const toasts = createToastStore();
```

- [ ] **Step 3: Create checks store in `ui/src/lib/stores/checks.ts`**

```ts
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
```

- [ ] **Step 4: Commit**

```bash
git add ui/src/lib/stores/
git commit -m "feat(ui): add auth, checks, and toast stores"
```

---

## Task 4: SSE Integration

**Files:**
- Create: `ui/src/lib/sse.ts`

- [ ] **Step 1: Create SSE client in `ui/src/lib/sse.ts`**

Connects to `/api/events`, parses events, and calls callbacks. EventSource handles reconnect automatically.

```ts
import type { CheckStatus } from './types';

interface SSEPayload {
  check_id: string;
  status: CheckStatus;
  event: string;
}

type SSECallback = (payload: SSEPayload) => void;

let eventSource: EventSource | null = null;
let callbacks: SSECallback[] = [];

export function connectSSE() {
  if (eventSource) return;

  eventSource = new EventSource('/api/events');

  eventSource.addEventListener('ping_received', (e) => dispatch(e));
  eventSource.addEventListener('status_changed', (e) => dispatch(e));
  eventSource.addEventListener('alert_fired', (e) => dispatch(e));

  eventSource.onerror = () => {
    // EventSource auto-reconnects; nothing to do here
  };
}

export function disconnectSSE() {
  eventSource?.close();
  eventSource = null;
}

export function onSSEEvent(callback: SSECallback): () => void {
  callbacks.push(callback);
  return () => {
    callbacks = callbacks.filter((cb) => cb !== callback);
  };
}

function dispatch(e: MessageEvent) {
  try {
    const payload: SSEPayload = JSON.parse(e.data);
    for (const cb of callbacks) {
      cb(payload);
    }
  } catch {
    // Ignore malformed events
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add ui/src/lib/sse.ts
git commit -m "feat(ui): add SSE EventSource client with event dispatch"
```

---

## Task 5: Shared Components

**Files:**
- Create: `ui/src/lib/components/StatusBadge.svelte`
- Create: `ui/src/lib/components/TimeAgo.svelte`
- Create: `ui/src/lib/components/Toast.svelte`
- Create: `ui/src/lib/components/ToastContainer.svelte`
- Create: `ui/src/lib/components/Modal.svelte`
- Create: `ui/src/lib/components/SkeletonCard.svelte`
- Create: `ui/src/lib/components/SkeletonRow.svelte`
- Create: `ui/src/lib/components/CheckCard.svelte`
- Create: `ui/src/lib/components/EmptyState.svelte`
- Create: `ui/src/lib/components/Nav.svelte`

- [ ] **Step 1: Create `StatusBadge.svelte`**

Per DESIGN.md: "Status badges are text + color, not just colored dots." Sharp 2px radius.

```svelte
<script lang="ts">
  import type { CheckStatus } from '$lib/types';

  interface Props {
    status: CheckStatus;
    size?: 'sm' | 'md';
  }

  let { status, size = 'md' }: Props = $props();

  const labels: Record<CheckStatus, string> = {
    new: 'NEW',
    up: 'UP',
    down: 'DOWN',
    alerting: 'ALERTING',
    silenced: 'SILENCED',
  };

  const colors: Record<CheckStatus, string> = {
    new: 'bg-status-new/20 text-status-new',
    up: 'bg-status-up/20 text-status-up',
    down: 'bg-status-down/20 text-status-down',
    alerting: 'bg-status-alerting/20 text-status-alerting',
    silenced: 'bg-status-silenced/20 text-status-silenced',
  };
</script>

<span
  class="inline-flex items-center font-mono font-medium rounded-badge {colors[status]} {size === 'sm' ? 'px-1.5 py-0.5 text-xs' : 'px-2 py-1 text-sm'}"
  aria-label="Status: {labels[status]}"
>
  {labels[status]}
</span>
```

- [ ] **Step 2: Create `TimeAgo.svelte`**

Per DESIGN.md: "Timestamps shown as human delta ('3h 12m ago') with ISO tooltip on hover."

```svelte
<script lang="ts">
  import { onMount } from 'svelte';

  interface Props {
    date: string | null;
    fallback?: string;
  }

  let { date, fallback = 'never' }: Props = $props();

  let display = $state(fallback);

  function format(iso: string): string {
    const diff = Date.now() - new Date(iso).getTime();
    const seconds = Math.floor(diff / 1000);
    if (seconds < 60) return `${seconds}s ago`;
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ${minutes % 60}m ago`;
    const days = Math.floor(hours / 24);
    return `${days}d ${hours % 24}h ago`;
  }

  function tick() {
    if (date) display = format(date);
  }

  onMount(() => {
    tick();
    const interval = setInterval(tick, 15000);
    return () => clearInterval(interval);
  });
</script>

{#if date}
  <time
    datetime={date}
    title={new Date(date).toLocaleString()}
    class="font-mono text-text-secondary"
  >
    {display}
  </time>
{:else}
  <span class="text-text-secondary">{fallback}</span>
{/if}
```

- [ ] **Step 3: Create `Toast.svelte` and `ToastContainer.svelte`**

`Toast.svelte`:
```svelte
<script lang="ts">
  import type { Toast as ToastType } from '$lib/stores/toast';
  import { toasts } from '$lib/stores/toast';

  interface Props {
    toast: ToastType;
  }

  let { toast }: Props = $props();

  const bgColors = {
    success: 'bg-status-up/10 border-status-up/30',
    error: 'bg-status-down/10 border-status-down/30',
    info: 'bg-surface border-border',
  };
</script>

<div
  class="flex items-center gap-3 px-4 py-3 rounded-card border {bgColors[toast.type]} text-sm"
  role="alert"
>
  <span class="flex-1">{toast.message}</span>
  <button
    class="text-text-secondary hover:text-text-primary"
    onclick={() => toasts.dismiss(toast.id)}
    aria-label="Dismiss"
  >
    &times;
  </button>
</div>
```

`ToastContainer.svelte`:
```svelte
<script lang="ts">
  import { toasts } from '$lib/stores/toast';
  import Toast from './Toast.svelte';
</script>

<div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2 w-80" aria-live="polite">
  {#each $toasts as toast (toast.id)}
    <Toast {toast} />
  {/each}
</div>
```

- [ ] **Step 4: Create `Modal.svelte`**

Per DESIGN.md: "focus trapped inside, Escape to close, role=dialog, aria-modal=true."

```svelte
<script lang="ts">
  import { onMount } from 'svelte';

  interface Props {
    title: string;
    open: boolean;
    onclose: () => void;
    children: any;
  }

  let { title, open = $bindable(), onclose, children }: Props = $props();

  let dialogEl: HTMLDialogElement;

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onclose();
    }
  }

  $effect(() => {
    if (open && dialogEl) {
      dialogEl.showModal();
    } else if (dialogEl?.open) {
      dialogEl.close();
    }
  });
</script>

{#if open}
  <dialog
    bind:this={dialogEl}
    class="bg-surface border border-border rounded-card p-0 text-text-primary backdrop:bg-black/60 max-w-md w-full"
    aria-labelledby="modal-title"
    aria-modal="true"
    onkeydown={handleKeydown}
    onclose={onclose}
  >
    <div class="p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 id="modal-title" class="text-lg font-semibold">{title}</h2>
        <button
          class="text-text-secondary hover:text-text-primary text-xl"
          onclick={onclose}
          aria-label="Close"
        >
          &times;
        </button>
      </div>
      {@render children()}
    </div>
  </dialog>
{/if}
```

- [ ] **Step 5: Create `SkeletonCard.svelte` and `SkeletonRow.svelte`**

Per DESIGN.md: "gray placeholder shapes at same dimensions as real content. No spinners on page load."

`SkeletonCard.svelte`:
```svelte
<div class="bg-surface border border-border rounded-card p-4 animate-pulse">
  <div class="flex items-center justify-between mb-3">
    <div class="h-5 w-32 bg-border rounded"></div>
    <div class="h-6 w-16 bg-border rounded-badge"></div>
  </div>
  <div class="h-4 w-24 bg-border rounded"></div>
</div>
```

`SkeletonRow.svelte`:
```svelte
<div class="flex items-center gap-4 py-3 px-4 animate-pulse">
  <div class="h-4 w-40 bg-border rounded"></div>
  <div class="h-4 w-24 bg-border rounded"></div>
  <div class="h-4 w-16 bg-border rounded"></div>
</div>
```

- [ ] **Step 6: Create `EmptyState.svelte`**

```svelte
<script lang="ts">
  interface Props {
    title: string;
    description?: string;
    children?: any;
  }

  let { title, description, children }: Props = $props();
</script>

<div class="flex flex-col items-center justify-center py-16 px-4 text-center">
  <h3 class="text-lg font-medium text-text-primary mb-2">{title}</h3>
  {#if description}
    <p class="text-text-secondary text-sm mb-6 max-w-md">{description}</p>
  {/if}
  {#if children}
    {@render children()}
  {/if}
</div>
```

- [ ] **Step 7: Create `CheckCard.svelte`**

Per DESIGN.md: "Failing checks show subtle left border accent (red), not full bg color."

```svelte
<script lang="ts">
  import type { Check } from '$lib/types';
  import StatusBadge from './StatusBadge.svelte';
  import TimeAgo from './TimeAgo.svelte';

  interface Props {
    check: Check;
  }

  let { check }: Props = $props();

  const borderColors: Record<string, string> = {
    up: 'border-l-status-up',
    down: 'border-l-status-down',
    alerting: 'border-l-status-alerting',
    new: 'border-l-status-new',
    silenced: 'border-l-status-silenced',
  };
</script>

<a
  href="/checks/{check.id}"
  class="block bg-surface border border-border {borderColors[check.status]} border-l-2 rounded-card p-4 hover:bg-border/30 transition-colors"
  role="article"
  aria-label="{check.name}, status {check.status}"
>
  <div class="flex items-center justify-between mb-2">
    <h3 class="font-medium truncate">{check.name}</h3>
    <StatusBadge status={check.status} size="sm" />
  </div>
  <div class="flex items-center gap-3 text-sm">
    <span class="text-text-secondary">Last ping:</span>
    <TimeAgo date={check.last_ping_at} fallback="waiting..." />
  </div>
</a>
```

- [ ] **Step 8: Create `Nav.svelte`**

Per DESIGN.md: "+ New Check is the ONLY primary action color." Mobile: bottom tab bar. Desktop: top nav.

```svelte
<script lang="ts">
  import { auth } from '$lib/stores/auth';
  import { page } from '$app/stores';

  const navItems = [
    { href: '/', label: 'Dashboard', icon: '&#9632;' },
    { href: '/alerts', label: 'Alerts', icon: '&#9888;' },
    { href: '/settings', label: 'Settings', icon: '&#9881;' },
  ];

  function isActive(href: string, pathname: string): boolean {
    if (href === '/') return pathname === '/';
    return pathname.startsWith(href);
  }
</script>

<!-- Desktop nav -->
<nav class="hidden md:flex items-center justify-between px-6 py-3 bg-surface border-b border-border">
  <div class="flex items-center gap-6">
    <a href="/" class="font-semibold text-text-primary hover:text-white">cronhealth</a>
    {#each navItems as item}
      <a
        href={item.href}
        class="text-sm {isActive(item.href, $page.url.pathname) ? 'text-text-primary' : 'text-text-secondary hover:text-text-primary'}"
      >
        {item.label}
      </a>
    {/each}
  </div>
  <div class="flex items-center gap-4">
    <a
      href="/checks/new"
      class="bg-status-up/20 text-status-up px-3 py-1.5 rounded-card text-sm font-medium hover:bg-status-up/30 transition-colors"
    >
      + New Check
    </a>
    {#if $auth.user}
      <span class="text-text-secondary text-sm">{$auth.user.email}</span>
    {/if}
  </div>
</nav>

<!-- Mobile bottom nav -->
<nav class="md:hidden fixed bottom-0 left-0 right-0 bg-surface border-t border-border flex items-center justify-around py-2 z-40">
  {#each navItems as item}
    <a
      href={item.href}
      class="flex flex-col items-center gap-0.5 px-3 py-1 text-xs {isActive(item.href, $page.url.pathname) ? 'text-text-primary' : 'text-text-secondary'}"
    >
      <span class="text-lg">{@html item.icon}</span>
      {item.label}
    </a>
  {/each}
  <a
    href="/checks/new"
    class="flex flex-col items-center gap-0.5 px-3 py-1 text-xs text-status-up"
  >
    <span class="text-lg">+</span>
    New
  </a>
</nav>
```

- [ ] **Step 9: Commit**

```bash
git add ui/src/lib/components/
git commit -m "feat(ui): add shared components (StatusBadge, CheckCard, Modal, Toast, Nav, etc.)"
```

---

## Task 6: Root Layout with Auth Guard and SSE

**Files:**
- Modify: `ui/src/routes/+layout.svelte`
- Modify: `ui/src/routes/+layout.ts`

- [ ] **Step 1: Update root layout**

Wire together: auth check, nav, toast container, SSE connection, and mobile bottom padding.

`ui/src/routes/+layout.svelte`:
```svelte
<script lang="ts">
  import '../app.css';
  import { onMount, onDestroy } from 'svelte';
  import { auth } from '$lib/stores/auth';
  import { checksStore } from '$lib/stores/checks';
  import { connectSSE, disconnectSSE, onSSEEvent } from '$lib/sse';
  import Nav from '$lib/components/Nav.svelte';
  import ToastContainer from '$lib/components/ToastContainer.svelte';

  let { children } = $props();

  onMount(async () => {
    await auth.load();
    await checksStore.load();
    connectSSE();
  });

  const unsubSSE = onSSEEvent((payload) => {
    checksStore.updateCheckStatus(payload.check_id, payload.status);
  });

  onDestroy(() => {
    disconnectSSE();
    unsubSSE();
  });
</script>

{#if $auth.loading}
  <div class="flex items-center justify-center h-screen">
    <span class="text-text-secondary">Loading...</span>
  </div>
{:else}
  <Nav />
  <main class="pb-16 md:pb-0" role="main">
    {@render children()}
  </main>
  <ToastContainer />
{/if}
```

`ui/src/routes/+layout.ts` (unchanged from Task 1):
```ts
export const prerender = false;
export const ssr = false;
```

- [ ] **Step 2: Commit**

```bash
git add ui/src/routes/+layout.svelte ui/src/routes/+layout.ts
git commit -m "feat(ui): wire root layout with auth, SSE, nav, and toasts"
```

---

## Task 7: Dashboard Page

**Files:**
- Modify: `ui/src/routes/+page.svelte`

- [ ] **Step 1: Build the dashboard page**

Per DESIGN.md hierarchy:
1. Status summary bar: "X checks OK, Y down, Z silenced"
2. Failing/alerting checks surfaced first
3. All other checks grid
4. First-run empty state when zero checks

```svelte
<script lang="ts">
  import { checksStore } from '$lib/stores/checks';
  import CheckCard from '$lib/components/CheckCard.svelte';
  import SkeletonCard from '$lib/components/SkeletonCard.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import type { Check } from '$lib/types';

  const failing = $derived(
    $checksStore.checks.filter((c: Check) => c.status === 'down' || c.status === 'alerting')
  );
  const healthy = $derived(
    $checksStore.checks.filter((c: Check) => c.status === 'up')
  );
  const other = $derived(
    $checksStore.checks.filter((c: Check) => c.status === 'new' || c.status === 'silenced')
  );

  const counts = $derived({
    total: $checksStore.checks.length,
    ok: healthy.length,
    down: failing.length,
    silenced: other.filter((c: Check) => c.status === 'silenced').length,
    new: other.filter((c: Check) => c.status === 'new').length,
  });
</script>

<svelte:head>
  <title>Dashboard - cronhealth</title>
</svelte:head>

<div class="max-w-5xl mx-auto px-4 py-6">
  {#if $checksStore.loading}
    <!-- Skeleton loading -->
    <div class="flex gap-4 mb-6">
      <div class="h-8 w-48 bg-border rounded animate-pulse"></div>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
      {#each Array(4) as _}
        <SkeletonCard />
      {/each}
    </div>
  {:else if counts.total === 0}
    <!-- First-run empty state -->
    <EmptyState
      title="No checks configured yet."
      description="Paste this into your first cron job to get started:"
    >
      <pre class="bg-surface border border-border rounded-card p-4 text-sm font-mono text-text-secondary mb-6 text-left">curl -fsS -X POST \
  http://cronhealth.internal/ping/YOUR-SLUG</pre>
      <a
        href="/checks/new"
        class="bg-status-up/20 text-status-up px-4 py-2 rounded-card font-medium hover:bg-status-up/30 transition-colors"
      >
        + Create your first check
      </a>
    </EmptyState>
  {:else}
    <!-- Status summary bar -->
    <div class="flex items-center gap-4 mb-6 text-sm" aria-live="polite">
      <span class="text-status-up font-medium">{counts.ok} OK</span>
      {#if counts.down > 0}
        <span class="text-status-down font-medium">{counts.down} DOWN</span>
      {/if}
      {#if counts.silenced > 0}
        <span class="text-status-silenced">{counts.silenced} silenced</span>
      {/if}
      {#if counts.new > 0}
        <span class="text-status-new">{counts.new} waiting</span>
      {/if}
    </div>

    <!-- Failing checks first -->
    {#if failing.length > 0}
      <section class="mb-6">
        <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">Failing</h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          {#each failing as check (check.id)}
            <CheckCard {check} />
          {/each}
        </div>
      </section>
    {/if}

    <!-- Healthy + other checks -->
    {#if healthy.length > 0 || other.length > 0}
      <section>
        <!-- Mobile: collapsible accordion for healthy checks -->
        <details class="md:hidden" open={failing.length === 0}>
          <summary class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide cursor-pointer">
            All Checks ({healthy.length + other.length})
          </summary>
          <div class="grid grid-cols-1 gap-3">
            {#each [...healthy, ...other] as check (check.id)}
              <CheckCard {check} />
            {/each}
          </div>
        </details>
        <!-- Desktop: always visible grid -->
        <div class="hidden md:block">
          <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">All Checks</h2>
          <div class="grid grid-cols-2 gap-3">
            {#each [...healthy, ...other] as check (check.id)}
              <CheckCard {check} />
            {/each}
          </div>
        </div>
      </section>
    {/if}
  {/if}
</div>
```

- [ ] **Step 2: Verify dev server renders dashboard**

```bash
cd ui && npm run dev -- --port 5173
```

Dashboard should render with skeleton cards (API calls will fail in dev without backend — that's expected).

- [ ] **Step 3: Commit**

```bash
git add ui/src/routes/+page.svelte
git commit -m "feat(ui): add dashboard page with status bar, failing-first layout, and empty state"
```

---

## Task 8: Check Detail Page

**Files:**
- Create: `ui/src/routes/checks/[id]/+page.svelte`

- [ ] **Step 1: Build check detail page**

Per DESIGN.md hierarchy:
1. Name + status badge (large)
2. Last ping time + next expected countdown
3. Action bar: Snooze / Silence / Edit / Delete
4. Ping URL (monospace, copy button)
5. Ping history timeline
6. Alert log (collapsible)

Snooze uses a modal with preset buttons (30m / 1h / 4h / 24h).

```svelte
<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { toasts } from '$lib/stores/toast';
  import { checksStore } from '$lib/stores/checks';
  import type { Check, Ping, Alert } from '$lib/types';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import TimeAgo from '$lib/components/TimeAgo.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import SkeletonRow from '$lib/components/SkeletonRow.svelte';

  let check: Check | null = $state(null);
  let pings: Ping[] = $state([]);
  let alerts: Alert[] = $state([]);
  let loading = $state(true);
  let pingsLoading = $state(true);
  let showSnoozeModal = $state(false);
  let showDeleteModal = $state(false);
  let actionLoading = $state(false);

  const checkId = $derived($page.params.id);
  const pingUrl = $derived(
    check ? `${window.location.origin}/ping/${check.slug}` : ''
  );

  onMount(async () => {
    try {
      check = await api.getCheck(checkId);
      loading = false;

      const [pingsResult, alertsResult] = await Promise.all([
        api.listPings(checkId),
        api.listAlerts(),
      ]);
      pings = pingsResult;
      alerts = alertsResult.filter((a) => a.check_id === checkId);
      pingsLoading = false;
    } catch {
      toasts.error('Failed to load check');
      loading = false;
      pingsLoading = false;
    }
  });

  async function handleSnooze(minutes: number) {
    actionLoading = true;
    try {
      await api.snoozeCheck(checkId, { duration_minutes: minutes });
      toasts.success(`Snoozed for ${minutes >= 60 ? `${minutes / 60}h` : `${minutes}m`}`);
      check = await api.getCheck(checkId);
      checksStore.updateCheck(check!);
      showSnoozeModal = false;
    } catch {
      toasts.error('Snooze failed');
    }
    actionLoading = false;
  }

  async function handleSilence() {
    actionLoading = true;
    try {
      await api.silenceCheck(checkId, {});
      toasts.success('Check silenced');
      check = await api.getCheck(checkId);
      checksStore.updateCheck(check!);
    } catch {
      toasts.error('Silence failed');
    }
    actionLoading = false;
  }

  async function handleRemoveSilence() {
    actionLoading = true;
    try {
      await api.removeSilence(checkId);
      toasts.success('Silence removed');
      check = await api.getCheck(checkId);
      checksStore.updateCheck(check!);
    } catch {
      toasts.error('Failed to remove silence');
    }
    actionLoading = false;
  }

  async function handleDelete() {
    actionLoading = true;
    try {
      await api.deleteCheck(checkId);
      checksStore.removeCheck(checkId);
      toasts.success('Check deleted');
      goto('/');
    } catch {
      toasts.error('Delete failed');
    }
    actionLoading = false;
  }

  function copyPingUrl() {
    navigator.clipboard.writeText(pingUrl);
    toasts.success('Ping URL copied');
  }
</script>

<svelte:head>
  <title>{check?.name ?? 'Check'} - cronhealth</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 py-6">
  {#if loading}
    <div class="animate-pulse">
      <div class="h-8 w-64 bg-border rounded mb-4"></div>
      <div class="h-5 w-40 bg-border rounded mb-6"></div>
    </div>
  {:else if check}
    <!-- Header: name + status -->
    <div class="flex items-start justify-between mb-4">
      <div>
        <h1 class="text-2xl font-semibold mb-2">{check.name}</h1>
        <StatusBadge status={check.status} />
      </div>
      <a
        href="/checks/{check.id}/edit"
        class="text-text-secondary hover:text-text-primary text-sm border border-border rounded-card px-3 py-1.5"
      >
        Edit
      </a>
    </div>

    <!-- Timing info -->
    <div class="flex flex-wrap gap-6 mb-6 text-sm">
      <div>
        <span class="text-text-secondary">Last ping:</span>
        <TimeAgo date={check.last_ping_at} fallback="waiting for first ping..." />
      </div>
      <div>
        <span class="text-text-secondary">Period:</span>
        <span class="font-mono">{check.period_seconds}s</span>
      </div>
      <div>
        <span class="text-text-secondary">Grace:</span>
        <span class="font-mono">{check.grace_seconds}s</span>
      </div>
    </div>

    <!-- Ping URL -->
    <div class="bg-surface border border-border rounded-card p-4 mb-6">
      <div class="text-sm text-text-secondary mb-2">Ping URL</div>
      <div class="flex items-center gap-2">
        <code class="font-mono text-sm text-text-primary flex-1 truncate">{pingUrl}</code>
        <button
          onclick={copyPingUrl}
          class="text-text-secondary hover:text-text-primary text-sm border border-border rounded-card px-2 py-1 shrink-0"
        >
          Copy
        </button>
      </div>
      <pre class="mt-3 text-xs text-text-secondary font-mono">curl -fsS -X POST {pingUrl}</pre>
    </div>

    <!-- Two-panel layout: info left, history right on desktop -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
    <div>
    <!-- Action bar -->
    <div class="flex flex-wrap gap-2 mb-8">
      <button
        onclick={() => (showSnoozeModal = true)}
        disabled={actionLoading}
        class="border border-border rounded-card px-3 py-1.5 text-sm hover:bg-border/30 transition-colors disabled:opacity-50"
      >
        Snooze
      </button>
      {#if check.status === 'silenced'}
        <button
          onclick={handleRemoveSilence}
          disabled={actionLoading}
          class="border border-border rounded-card px-3 py-1.5 text-sm hover:bg-border/30 transition-colors disabled:opacity-50"
        >
          Remove Silence
        </button>
      {:else}
        <button
          onclick={handleSilence}
          disabled={actionLoading}
          class="border border-border rounded-card px-3 py-1.5 text-sm hover:bg-border/30 transition-colors disabled:opacity-50"
        >
          Silence
        </button>
      {/if}
      <button
        onclick={() => (showDeleteModal = true)}
        disabled={actionLoading}
        class="border border-status-down/30 text-status-down rounded-card px-3 py-1.5 text-sm hover:bg-status-down/10 transition-colors disabled:opacity-50"
      >
        Delete
      </button>
    </div>

    </div><!-- end left panel -->
    <div>
    <!-- Ping history -->
    <section class="mb-8">
      <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">Ping History</h2>
      {#if pingsLoading}
        {#each Array(5) as _}
          <SkeletonRow />
        {/each}
      {:else if pings.length === 0}
        <p class="text-text-secondary text-sm py-4">No pings yet — waiting for first ping.</p>
      {:else}
        <div class="bg-surface border border-border rounded-card divide-y divide-border">
          {#each pings as ping (ping.id)}
            <div class="flex items-center justify-between px-4 py-2.5 text-sm">
              <TimeAgo date={ping.received_at} />
              <div class="flex items-center gap-4">
                {#if ping.source_ip}
                  <span class="font-mono text-text-secondary text-xs hidden md:inline">{ping.source_ip}</span>
                {/if}
                {#if ping.exit_code !== null}
                  <span class="font-mono text-xs {ping.exit_code === 0 ? 'text-status-up' : 'text-status-down'}">
                    exit {ping.exit_code}
                  </span>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </section>

    <!-- Alert log -->
    {#if alerts.length > 0}
      <section>
        <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">Alert History</h2>
        <div class="bg-surface border border-border rounded-card divide-y divide-border">
          {#each alerts as alert (alert.id)}
            <div class="flex items-center justify-between px-4 py-2.5 text-sm">
              <div class="flex items-center gap-3">
                <span class={alert.resolved_at ? 'text-text-secondary' : 'text-status-down font-medium'}>
                  {alert.resolved_at ? 'Resolved' : 'Active'}
                </span>
                <TimeAgo date={alert.started_at} />
              </div>
              <span class="font-mono text-text-secondary text-xs">{alert.alert_count} notification{alert.alert_count !== 1 ? 's' : ''}</span>
            </div>
          {/each}
        </div>
      </section>
    {/if}

    </div><!-- end right panel -->
    </div><!-- end two-panel grid -->

    <!-- Snooze modal -->
    <Modal title="Snooze Check" bind:open={showSnoozeModal} onclose={() => (showSnoozeModal = false)}>
      <p class="text-text-secondary text-sm mb-4">Suppress notifications for:</p>
      <div class="grid grid-cols-2 gap-2">
        {#each [30, 60, 240, 1440] as minutes}
          <button
            onclick={() => handleSnooze(minutes)}
            disabled={actionLoading}
            class="border border-border rounded-card py-3 text-sm font-medium hover:bg-border/30 transition-colors disabled:opacity-50"
          >
            {minutes >= 60 ? `${minutes / 60}h` : `${minutes}m`}
          </button>
        {/each}
      </div>
    </Modal>

    <!-- Delete confirm modal -->
    <Modal title="Delete Check" bind:open={showDeleteModal} onclose={() => (showDeleteModal = false)}>
      <p class="text-text-secondary text-sm mb-4">
        Delete <strong class="text-text-primary">{check.name}</strong>? This removes all ping history and cannot be undone.
      </p>
      <div class="flex gap-2 justify-end">
        <button
          onclick={() => (showDeleteModal = false)}
          class="border border-border rounded-card px-4 py-2 text-sm hover:bg-border/30 transition-colors"
        >
          Cancel
        </button>
        <button
          onclick={handleDelete}
          disabled={actionLoading}
          class="bg-status-down/20 text-status-down border border-status-down/30 rounded-card px-4 py-2 text-sm hover:bg-status-down/30 transition-colors disabled:opacity-50"
        >
          Delete
        </button>
      </div>
    </Modal>
  {:else}
    <p class="text-text-secondary">Check not found.</p>
  {/if}
</div>
```

- [ ] **Step 2: Commit**

```bash
git add ui/src/routes/checks/
git commit -m "feat(ui): add check detail page with ping history, actions, and snooze modal"
```

---

## Task 9: New / Edit Check Forms

**Files:**
- Create: `ui/src/routes/checks/new/+page.svelte`
- Create: `ui/src/routes/checks/[id]/edit/+page.svelte`

- [ ] **Step 1: Create new check form at `ui/src/routes/checks/new/+page.svelte`**

Per DESIGN.md: "Minimal form — only required fields. URL shown immediately after save."

```svelte
<script lang="ts">
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { toasts } from '$lib/stores/toast';
  import { checksStore } from '$lib/stores/checks';

  let name = $state('');
  let periodMinutes = $state(5);
  let graceMinutes = $state(5);
  let submitting = $state(false);
  let error = $state('');

  async function handleSubmit() {
    error = '';
    if (!name.trim()) {
      error = 'Name is required';
      return;
    }
    if (periodMinutes < 1) {
      error = 'Period must be at least 1 minute';
      return;
    }

    submitting = true;
    try {
      const check = await api.createCheck({
        name: name.trim(),
        period_seconds: periodMinutes * 60,
        grace_seconds: graceMinutes * 60,
      });
      checksStore.addCheck(check);
      toasts.success('Check created');
      goto(`/checks/${check.id}`);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to create check';
    }
    submitting = false;
  }
</script>

<svelte:head>
  <title>New Check - cronhealth</title>
</svelte:head>

<div class="max-w-lg mx-auto px-4 py-6">
  <h1 class="text-2xl font-semibold mb-6">New Check</h1>

  <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-4">
    <div>
      <label for="name" class="block text-sm text-text-secondary mb-1">Name</label>
      <input
        id="name"
        type="text"
        bind:value={name}
        placeholder="e.g. nightly-backup"
        class="w-full bg-surface border border-border rounded-card px-3 py-2 text-sm text-text-primary placeholder:text-text-secondary/50 focus:outline-none focus:border-status-up/50"
      />
    </div>

    <div>
      <label for="period" class="block text-sm text-text-secondary mb-1">Expected period (minutes)</label>
      <input
        id="period"
        type="number"
        bind:value={periodMinutes}
        min="1"
        inputmode="numeric"
        class="w-full bg-surface border border-border rounded-card px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-status-up/50"
      />
      <p class="text-xs text-text-secondary mt-1">How often should this job ping?</p>
    </div>

    <div>
      <label for="grace" class="block text-sm text-text-secondary mb-1">Grace period (minutes)</label>
      <input
        id="grace"
        type="number"
        bind:value={graceMinutes}
        min="1"
        inputmode="numeric"
        class="w-full bg-surface border border-border rounded-card px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-status-up/50"
      />
      <p class="text-xs text-text-secondary mt-1">How long to wait after a missed ping before alerting.</p>
    </div>

    {#if error}
      <p class="text-status-down text-sm">{error}</p>
    {/if}

    <div class="flex gap-3 pt-2">
      <button
        type="submit"
        disabled={submitting}
        class="bg-status-up/20 text-status-up px-4 py-2 rounded-card text-sm font-medium hover:bg-status-up/30 transition-colors disabled:opacity-50"
      >
        {submitting ? 'Creating...' : 'Create Check'}
      </button>
      <a
        href="/"
        class="border border-border rounded-card px-4 py-2 text-sm text-text-secondary hover:text-text-primary transition-colors"
      >
        Cancel
      </a>
    </div>
  </form>
</div>
```

- [ ] **Step 2: Create edit check form at `ui/src/routes/checks/[id]/edit/+page.svelte`**

```svelte
<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { toasts } from '$lib/stores/toast';
  import { checksStore } from '$lib/stores/checks';
  import type { Check } from '$lib/types';

  let check: Check | null = $state(null);
  let name = $state('');
  let periodMinutes = $state(5);
  let graceMinutes = $state(5);
  let submitting = $state(false);
  let loading = $state(true);
  let error = $state('');

  const checkId = $derived($page.params.id);

  onMount(async () => {
    try {
      check = await api.getCheck(checkId);
      if (check) {
        name = check.name;
        periodMinutes = Math.round(check.period_seconds / 60);
        graceMinutes = Math.round(check.grace_seconds / 60);
      }
    } catch {
      toasts.error('Failed to load check');
    }
    loading = false;
  });

  async function handleSubmit() {
    error = '';
    if (!name.trim()) {
      error = 'Name is required';
      return;
    }

    submitting = true;
    try {
      const updated = await api.updateCheck(checkId, {
        name: name.trim(),
        period_seconds: periodMinutes * 60,
        grace_seconds: graceMinutes * 60,
      });
      if (updated) checksStore.updateCheck(updated);
      toasts.success('Check updated');
      goto(`/checks/${checkId}`);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to update check';
    }
    submitting = false;
  }
</script>

<svelte:head>
  <title>Edit {check?.name ?? 'Check'} - cronhealth</title>
</svelte:head>

<div class="max-w-lg mx-auto px-4 py-6">
  <h1 class="text-2xl font-semibold mb-6">Edit Check</h1>

  {#if loading}
    <div class="animate-pulse space-y-4">
      <div class="h-10 w-full bg-border rounded"></div>
      <div class="h-10 w-full bg-border rounded"></div>
      <div class="h-10 w-full bg-border rounded"></div>
    </div>
  {:else if check}
    <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-4">
      <div>
        <label for="name" class="block text-sm text-text-secondary mb-1">Name</label>
        <input
          id="name"
          type="text"
          bind:value={name}
          class="w-full bg-surface border border-border rounded-card px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-status-up/50"
        />
      </div>

      <div>
        <label for="period" class="block text-sm text-text-secondary mb-1">Expected period (minutes)</label>
        <input
          id="period"
          type="number"
          bind:value={periodMinutes}
          min="1"
          inputmode="numeric"
          class="w-full bg-surface border border-border rounded-card px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-status-up/50"
        />
      </div>

      <div>
        <label for="grace" class="block text-sm text-text-secondary mb-1">Grace period (minutes)</label>
        <input
          id="grace"
          type="number"
          bind:value={graceMinutes}
          min="1"
          inputmode="numeric"
          class="w-full bg-surface border border-border rounded-card px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-status-up/50"
        />
      </div>

      {#if error}
        <p class="text-status-down text-sm">{error}</p>
      {/if}

      <div class="flex gap-3 pt-2">
        <button
          type="submit"
          disabled={submitting}
          class="bg-status-up/20 text-status-up px-4 py-2 rounded-card text-sm font-medium hover:bg-status-up/30 transition-colors disabled:opacity-50"
        >
          {submitting ? 'Saving...' : 'Save Changes'}
        </button>
        <a
          href="/checks/{checkId}"
          class="border border-border rounded-card px-4 py-2 text-sm text-text-secondary hover:text-text-primary transition-colors"
        >
          Cancel
        </a>
      </div>
    </form>
  {:else}
    <p class="text-text-secondary">Check not found.</p>
  {/if}
</div>
```

- [ ] **Step 3: Commit**

```bash
git add ui/src/routes/checks/
git commit -m "feat(ui): add new and edit check form pages"
```

---

## Task 10: Alerts Feed Page

**Files:**
- Create: `ui/src/routes/alerts/+page.svelte`

- [ ] **Step 1: Build alerts feed page**

Per DESIGN.md: Active alerts (top, red), recently resolved (muted).

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { toasts } from '$lib/stores/toast';
  import { checksStore } from '$lib/stores/checks';
  import type { Alert } from '$lib/types';
  import TimeAgo from '$lib/components/TimeAgo.svelte';
  import SkeletonRow from '$lib/components/SkeletonRow.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';

  let alerts: Alert[] = $state([]);
  let loading = $state(true);
  let actionLoading = $state<string | null>(null);

  const active = $derived(alerts.filter((a) => !a.resolved_at));
  const resolved = $derived(alerts.filter((a) => a.resolved_at));

  async function handleSnooze(checkId: string, minutes: number) {
    actionLoading = checkId;
    try {
      await api.snoozeCheck(checkId, { duration_minutes: minutes });
      toasts.success(`Snoozed for ${minutes >= 60 ? `${minutes / 60}h` : `${minutes}m`}`);
      checksStore.updateCheckStatus(checkId, 'silenced');
      alerts = await api.listAlerts();
    } catch {
      toasts.error('Snooze failed');
    }
    actionLoading = null;
  }

  onMount(async () => {
    try {
      alerts = await api.listAlerts();
    } catch {
      toasts.error('Failed to load alerts');
    }
    loading = false;
  });
</script>

<svelte:head>
  <title>Alerts - cronhealth</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 py-6">
  <h1 class="text-2xl font-semibold mb-6">Alerts</h1>

  {#if loading}
    {#each Array(5) as _}
      <SkeletonRow />
    {/each}
  {:else if alerts.length === 0}
    <EmptyState
      title="No alerts"
      description="Everything looks healthy. No active or recent alerts."
    />
  {:else}
    {#if active.length > 0}
      <section class="mb-8">
        <h2 class="text-sm font-medium text-status-down mb-3 uppercase tracking-wide">Active</h2>
        <div class="bg-surface border border-border rounded-card divide-y divide-border">
          {#each active as alert (alert.id)}
            <a
              href="/checks/{alert.check_id}"
              class="flex items-center justify-between px-4 py-3 text-sm hover:bg-border/30 transition-colors"
            >
              <div class="flex items-center gap-3">
                <span class="text-status-down font-medium">{alert.check_name}</span>
                <span class="text-text-secondary">firing since</span>
                <TimeAgo date={alert.started_at} />
              </div>
              <div class="flex items-center gap-2">
                <span class="font-mono text-text-secondary text-xs">{alert.alert_count} sent</span>
                <button
                  onclick={(e) => { e.stopPropagation(); e.preventDefault(); handleSnooze(alert.check_id, 60); }}
                  disabled={actionLoading === alert.check_id}
                  class="border border-border rounded-badge px-2 py-0.5 text-xs text-text-secondary hover:text-text-primary transition-colors disabled:opacity-50"
                >
                  Snooze 1h
                </button>
              </div>
            </a>
          {/each}
        </div>
      </section>
    {/if}

    {#if resolved.length > 0}
      <section>
        <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">Resolved</h2>
        <div class="bg-surface border border-border rounded-card divide-y divide-border">
          {#each resolved as alert (alert.id)}
            <a
              href="/checks/{alert.check_id}"
              class="flex items-center justify-between px-4 py-3 text-sm hover:bg-border/30 transition-colors text-text-secondary"
            >
              <div class="flex items-center gap-3">
                <span>{alert.check_name}</span>
                <TimeAgo date={alert.resolved_at} />
              </div>
              <span class="font-mono text-xs">{alert.alert_count} sent</span>
            </a>
          {/each}
        </div>
      </section>
    {/if}
  {/if}
</div>
```

- [ ] **Step 2: Commit**

```bash
git add ui/src/routes/alerts/
git commit -m "feat(ui): add alerts feed page with active/resolved sections"
```

---

## Task 11: Settings Page

**Files:**
- Create: `ui/src/routes/settings/+page.svelte`

- [ ] **Step 1: Build settings page**

Per DESIGN.md: User profile, notification channel management.

```svelte
<script lang="ts">
  import { auth } from '$lib/stores/auth';
  import { toasts } from '$lib/stores/toast';
</script>

<svelte:head>
  <title>Settings - cronhealth</title>
</svelte:head>

<div class="max-w-lg mx-auto px-4 py-6">
  <h1 class="text-2xl font-semibold mb-6">Settings</h1>

  <!-- Profile -->
  <section class="mb-8">
    <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">Profile</h2>
    <div class="bg-surface border border-border rounded-card p-4">
      {#if $auth.user}
        <div class="text-sm">
          <div class="flex justify-between py-2">
            <span class="text-text-secondary">Email</span>
            <span class="font-mono">{$auth.user.email}</span>
          </div>
          <div class="flex justify-between py-2">
            <span class="text-text-secondary">User ID</span>
            <span class="font-mono text-xs text-text-secondary">{$auth.user.user_id}</span>
          </div>
        </div>
      {/if}
    </div>
  </section>

  <!-- Logout -->
  <section>
    <form
      method="POST"
      action="/auth/logout"
      onsubmit={() => {
        auth.logout();
        toasts.info('Logged out');
      }}
    >
      <button
        type="submit"
        class="border border-border rounded-card px-4 py-2 text-sm text-text-secondary hover:text-text-primary transition-colors"
      >
        Log out
      </button>
    </form>
  </section>
</div>
```

- [ ] **Step 2: Commit**

```bash
git add ui/src/routes/settings/
git commit -m "feat(ui): add settings page with profile info and logout"
```

---

## Known Gaps (Backend Prerequisites)

The following DESIGN.md features require backend API endpoints that don't exist yet:

1. **Notification channel CRUD** — The settings page should let users create/edit/delete notification channels (email/SMS). Needs: `GET/POST /api/channels`, `PUT/DELETE /api/channels/:id`.
2. **Channel selection in check forms** — New/edit check forms should include a multi-select for notification channels. The `CreateCheckRequest.channel_ids` field exists but has no UI because there's no way to list available channels yet.

These will be added as a backend follow-up task before the frontend can implement them.

---

## Task 12: Build Verification and Cleanup

**Files:**
- Create: `ui/static/favicon.svg`

- [ ] **Step 1: Add a minimal favicon**

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
  <rect width="32" height="32" rx="4" fill="#0f1117"/>
  <circle cx="16" cy="16" r="8" fill="none" stroke="#22c55e" stroke-width="2"/>
  <path d="M12 16l3 3 5-6" fill="none" stroke="#22c55e" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
</svg>
```

- [ ] **Step 2: Verify full build succeeds**

```bash
cd ui && npm run build
```

Should produce `ui/build/` with `index.html` fallback and all assets.

- [ ] **Step 3: Run type check**

```bash
cd ui && npx svelte-check --tsconfig ./tsconfig.json
```

Fix any type errors.

- [ ] **Step 4: Commit**

```bash
git add ui/static/favicon.svg
git commit -m "feat(ui): add favicon and verify clean build"
```

---

## Task 13: Update IMPLEMENTATION.md

**Files:**
- Modify: `IMPLEMENTATION.md`

- [ ] **Step 1: Update implementation log**

Move the SvelteKit Frontend section from "Not Started" to "Completed" with file list and descriptions.

- [ ] **Step 2: Commit**

```bash
git add IMPLEMENTATION.md
git commit -m "docs: mark SvelteKit frontend as complete in implementation log"
```
