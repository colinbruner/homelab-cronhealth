import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// We need to test the ApiClient class. Since it's instantiated at module scope
// as a singleton, we'll re-import after setting up mocks.

// Mock window.location
const mockLocation = { href: '' };
vi.stubGlobal('location', mockLocation);

// Mock fetch
const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

// Import after mocking globals
import { api } from './api';

function jsonResponse(data: unknown, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? 'OK' : 'Error',
    json: () => Promise.resolve(data),
  });
}

describe('ApiClient', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockLocation.href = '';
  });

  describe('listChecks', () => {
    it('fetches checks from /api/checks', async () => {
      const checks = [{ id: '1', name: 'test', status: 'up' }];
      mockFetch.mockReturnValue(jsonResponse(checks));

      const result = await api.listChecks();
      expect(result).toEqual(checks);
      expect(mockFetch).toHaveBeenCalledWith('/api/checks', expect.objectContaining({
        headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
      }));
    });
  });

  describe('getCheck', () => {
    it('fetches a single check by ID', async () => {
      const check = { id: 'abc', name: 'backup' };
      mockFetch.mockReturnValue(jsonResponse(check));

      const result = await api.getCheck('abc');
      expect(result).toEqual(check);
      expect(mockFetch).toHaveBeenCalledWith('/api/checks/abc', expect.any(Object));
    });
  });

  describe('createCheck', () => {
    it('posts check data', async () => {
      const newCheck = { id: '1', name: 'new-check', status: 'new' };
      mockFetch.mockReturnValue(jsonResponse(newCheck, 201));

      const result = await api.createCheck({
        name: 'new-check',
        period_seconds: 3600,
      });

      expect(result).toEqual(newCheck);
      expect(mockFetch).toHaveBeenCalledWith('/api/checks', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ name: 'new-check', period_seconds: 3600 }),
      }));
    });
  });

  describe('updateCheck', () => {
    it('puts updated check data', async () => {
      const updated = { id: '1', name: 'renamed' };
      mockFetch.mockReturnValue(jsonResponse(updated));

      await api.updateCheck('1', { name: 'renamed', period_seconds: 7200 });

      expect(mockFetch).toHaveBeenCalledWith('/api/checks/1', expect.objectContaining({
        method: 'PUT',
      }));
    });
  });

  describe('deleteCheck', () => {
    it('sends DELETE request', async () => {
      mockFetch.mockReturnValue(jsonResponse({ ok: true }));

      const result = await api.deleteCheck('xyz');
      expect(result).toEqual({ ok: true });
      expect(mockFetch).toHaveBeenCalledWith('/api/checks/xyz', expect.objectContaining({
        method: 'DELETE',
      }));
    });
  });

  describe('listPings', () => {
    it('includes limit and offset params', async () => {
      mockFetch.mockReturnValue(jsonResponse([]));

      await api.listPings('check-1', 25, 10);
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/checks/check-1/pings?limit=25&offset=10',
        expect.any(Object),
      );
    });

    it('uses default limit and offset', async () => {
      mockFetch.mockReturnValue(jsonResponse([]));

      await api.listPings('check-1');
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/checks/check-1/pings?limit=50&offset=0',
        expect.any(Object),
      );
    });
  });

  describe('snoozeCheck', () => {
    it('posts snooze request', async () => {
      mockFetch.mockReturnValue(jsonResponse({ id: 's1' }));

      await api.snoozeCheck('c1', { duration_minutes: 30 });
      expect(mockFetch).toHaveBeenCalledWith('/api/checks/c1/snooze', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ duration_minutes: 30 }),
      }));
    });
  });

  describe('silenceCheck', () => {
    it('posts silence request', async () => {
      mockFetch.mockReturnValue(jsonResponse({ id: 's2' }));

      await api.silenceCheck('c1', { reason: 'maintenance' });
      expect(mockFetch).toHaveBeenCalledWith('/api/checks/c1/silence', expect.objectContaining({
        method: 'POST',
      }));
    });
  });

  describe('removeSilence', () => {
    it('sends DELETE to silence endpoint', async () => {
      mockFetch.mockReturnValue(jsonResponse({ ok: true }));

      await api.removeSilence('c1');
      expect(mockFetch).toHaveBeenCalledWith('/api/checks/c1/silence', expect.objectContaining({
        method: 'DELETE',
      }));
    });
  });

  describe('listAlerts', () => {
    it('fetches from /api/alerts', async () => {
      mockFetch.mockReturnValue(jsonResponse([]));

      await api.listAlerts();
      expect(mockFetch).toHaveBeenCalledWith('/api/alerts', expect.any(Object));
    });
  });

  describe('me', () => {
    it('fetches current user', async () => {
      const user = { user_id: '1', email: 'test@example.com' };
      mockFetch.mockReturnValue(jsonResponse(user));

      const result = await api.me();
      expect(result).toEqual(user);
    });
  });

  describe('error handling', () => {
    it('redirects to login on 401', async () => {
      mockFetch.mockReturnValue(jsonResponse({ error: 'unauthorized' }, 401));

      await expect(api.listChecks()).rejects.toThrow('Authentication required');
      expect(mockLocation.href).toBe('/auth/login');
    });

    it('throws with error message from response body', async () => {
      mockFetch.mockReturnValue(jsonResponse({ error: 'check not found' }, 404));

      await expect(api.getCheck('bad-id')).rejects.toThrow('check not found');
    });

    it('falls back to statusText when body has no error field', async () => {
      mockFetch.mockReturnValue(Promise.resolve({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: () => Promise.resolve({}),
      }));

      await expect(api.listChecks()).rejects.toThrow('Internal Server Error');
    });
  });
});
