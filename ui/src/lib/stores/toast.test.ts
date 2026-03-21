import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { toasts } from './toast';

// Mock crypto.randomUUID
let uuidCounter = 0;
vi.stubGlobal('crypto', {
  randomUUID: () => `uuid-${++uuidCounter}`,
});

describe('toast store', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    uuidCounter = 0;
    // Clear any existing toasts
    const current = get(toasts);
    current.forEach((t) => toasts.dismiss(t.id));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('starts empty', () => {
    expect(get(toasts)).toEqual([]);
  });

  it('adds a success toast', () => {
    toasts.success('Check created');
    const items = get(toasts);
    expect(items).toHaveLength(1);
    expect(items[0].message).toBe('Check created');
    expect(items[0].type).toBe('success');
  });

  it('adds an error toast', () => {
    toasts.error('Failed to save');
    const items = get(toasts);
    expect(items).toHaveLength(1);
    expect(items[0].type).toBe('error');
  });

  it('adds an info toast', () => {
    toasts.info('Refreshing...');
    const items = get(toasts);
    expect(items).toHaveLength(1);
    expect(items[0].type).toBe('info');
  });

  it('auto-dismisses after 4 seconds', () => {
    toasts.success('Temporary');
    expect(get(toasts)).toHaveLength(1);

    vi.advanceTimersByTime(4000);
    expect(get(toasts)).toHaveLength(0);
  });

  it('manually dismisses a toast', () => {
    toasts.success('Dismissable');
    const id = get(toasts)[0].id;
    toasts.dismiss(id);
    expect(get(toasts)).toHaveLength(0);
  });

  it('supports multiple concurrent toasts', () => {
    toasts.success('First');
    toasts.error('Second');
    toasts.info('Third');
    expect(get(toasts)).toHaveLength(3);
  });

  it('assigns unique IDs', () => {
    toasts.success('A');
    toasts.success('B');
    const items = get(toasts);
    expect(items[0].id).not.toBe(items[1].id);
  });
});
