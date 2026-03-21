import type {
  Check, Ping, Alert, Silence, User,
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

  listPings(checkId: string, limit = 50, offset = 0): Promise<Ping[]> {
    return this.request(`/api/checks/${checkId}/pings?limit=${limit}&offset=${offset}`);
  }

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

  listAlerts(): Promise<Alert[]> {
    return this.request('/api/alerts');
  }

  getAlert(id: string): Promise<Alert> {
    return this.request(`/api/alerts/${id}`);
  }

  me(): Promise<User> {
    return this.request('/api/me');
  }
}

export const api = new ApiClient();
