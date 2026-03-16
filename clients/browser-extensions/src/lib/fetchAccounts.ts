import type { Account } from '../types';
import { sendMessage } from './apiClient';

/**
 * Fetch all accounts and the active account ID from the background service worker.
 * Shared between PopupApp and OptionsApp to avoid duplication.
 */
export async function fetchAccounts(): Promise<{ accounts: Account[]; activeId: string | null }> {
  const res = await sendMessage<Account[]>({ type: 'GET_ACCOUNTS' });
  const accounts = res.success && res.data ? res.data : [];
  const storage = await chrome.storage.local.get('activeAccountId');
  const activeId = (storage['activeAccountId'] as string | null | undefined) ?? null;
  return { accounts, activeId };
}
