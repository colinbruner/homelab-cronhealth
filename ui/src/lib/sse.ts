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
