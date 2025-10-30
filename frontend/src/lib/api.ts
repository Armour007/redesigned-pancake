export const API_BASE = (import.meta as any).env?.PUBLIC_API_BASE || 'http://localhost:8081';

export function authHeaders(token: string) {
  return {
    Authorization: `Bearer ${token}`
  } as Record<string, string>;
}
