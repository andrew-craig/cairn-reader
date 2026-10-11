import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import type { UnifiedSubscription } from '@cairn/shared';
import Rss from './Rss';
import Newsletters from './Newsletters';
import { ReadService } from '../services/read';

const rssSub: UnifiedSubscription = {
  id: 'sub-rss',
  type: 'rss',
  title: 'Hacker News',
  subscribed_at: '2026-01-01T00:00:00Z',
  list: 'feed',
  rss_data: { feed_id: 'feed-1', feed_url: 'https://news.example/rss' } as UnifiedSubscription['rss_data'],
};

const emailSub: UnifiedSubscription = {
  id: 'sender-1',
  type: 'email',
  title: 'Weekly Letter',
  subscribed_at: '2026-01-01T00:00:00Z',
  list: 'reads',
  email_data: { email_address: 'letter@example.com' } as UnifiedSubscription['email_data'],
};

describe('subscription screens', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(ReadService, 'listAllSubscriptions').mockResolvedValue({
      subscriptions: [rssSub, emailSub],
      total_count: 2,
      failed_sources: [],
    });
    vi.spyOn(ReadService, 'getOrCreateEmailAddress').mockResolvedValue('me@cairn.test');
  });

  it('RSS lists only feeds, each with a Feed/Reads toggle and the new-items note', async () => {
    render(<Rss />);

    await screen.findByText('Hacker News');
    expect(screen.queryByText('Weekly Letter')).toBeNull();
    expect(screen.getByRole('group', { name: 'Destination for Hacker News' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Feed' }).getAttribute('aria-pressed')).toBe('true');
    expect(screen.getByText(/applies to new items only/i)).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Unsubscribe' })).toBeTruthy();
  });

  it('Newsletters lists only senders, each with a toggle and no unsubscribe', async () => {
    render(<Newsletters />);

    await screen.findByText('Weekly Letter');
    expect(screen.queryByText('Hacker News')).toBeNull();
    expect(screen.getByRole('group', { name: 'Destination for Weekly Letter' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Reads' }).getAttribute('aria-pressed')).toBe('true');
    expect(screen.queryByRole('button', { name: 'Unsubscribe' })).toBeNull();
  });
});
