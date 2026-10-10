import React from 'react';
import { Alert } from 'react-native';
import { render, screen, fireEvent, waitFor } from '@testing-library/react-native';
import type { UnifiedSubscription } from '@cairn/shared';
import { SubscriptionListScreen } from './SubscriptionListScreen';
import { ReadService } from '../services/read';

jest.mock('../services/read', () => ({
  ReadService: {
    listAllSubscriptions: jest.fn(),
    setSourceList: jest.fn(),
    unsubscribeFromRSSFeed: jest.fn(),
  },
}));

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ canGoBack: () => true, goBack: jest.fn() }),
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  useFocusEffect: (cb: () => void) => require('react').useEffect(() => cb(), [cb]),
}));

const mockedReadService = ReadService as jest.Mocked<typeof ReadService>;

const rss = (overrides: Partial<UnifiedSubscription> = {}): UnifiedSubscription => ({
  id: 'sub-rss',
  type: 'rss',
  title: 'Go Blog',
  subscribed_at: '2025-01-01T00:00:00Z',
  list: 'reads',
  rss_data: { feed_id: 'feed-1', feed_url: 'https://go.dev/feed' },
  ...overrides,
});

const email = (overrides: Partial<UnifiedSubscription> = {}): UnifiedSubscription => ({
  id: 'sender-1',
  type: 'email',
  title: 'Weekly Letter',
  subscribed_at: '2025-01-01T00:00:00Z',
  list: 'reads',
  email_data: { email_address: 'letter@example.com' },
  ...overrides,
});

const stable = { filter: () => true };

function renderList(subs: UnifiedSubscription[]) {
  mockedReadService.listAllSubscriptions.mockResolvedValue({
    subscriptions: subs,
    total_count: subs.length,
    failed_sources: [],
  });
  return render(<SubscriptionListScreen title="Sources" filter={stable.filter} />);
}

const feedButton = (title: string) => screen.getByLabelText(`Send ${title} to Feed`);
const readsButton = (title: string) => screen.getByLabelText(`Send ${title} to Reads`);
const isSelected = (el: { props: { accessibilityState?: { selected?: boolean } } }) =>
  el.props.accessibilityState?.selected === true;

describe('SubscriptionListScreen Feed/Reads routing', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => jest.restoreAllMocks());

  it("shows each source's current list and says the choice only affects new items", async () => {
    renderList([rss({ list: 'feed' }), email({ list: 'reads' })]);
    await screen.findByText('Go Blog');

    expect(isSelected(feedButton('Go Blog'))).toBe(true);
    expect(isSelected(readsButton('Go Blog'))).toBe(false);
    expect(isSelected(readsButton('Weekly Letter'))).toBe(true);
    expect(screen.getByText(/new items only/i)).toBeTruthy();
  });

  it('routes an RSS feed by feed_id and reflects the change immediately', async () => {
    mockedReadService.setSourceList.mockResolvedValue({ type: 'rss', key: 'feed-1', list: 'feed' });
    renderList([rss()]);
    await screen.findByText('Go Blog');

    fireEvent.press(feedButton('Go Blog'));

    expect(isSelected(feedButton('Go Blog'))).toBe(true);
    await waitFor(() =>
      expect(mockedReadService.setSourceList).toHaveBeenCalledWith('rss', 'feed-1', 'feed'),
    );
  });

  it('routes an email sender by its subscription id', async () => {
    mockedReadService.setSourceList.mockResolvedValue({ type: 'email', key: 'sender-1', list: 'feed' });
    renderList([email()]);
    await screen.findByText('Weekly Letter');

    fireEvent.press(feedButton('Weekly Letter'));

    await waitFor(() =>
      expect(mockedReadService.setSourceList).toHaveBeenCalledWith('email', 'sender-1', 'feed'),
    );
  });

  it('does nothing when the already-selected list is pressed', async () => {
    renderList([rss({ list: 'reads' })]);
    await screen.findByText('Go Blog');

    fireEvent.press(readsButton('Go Blog'));

    expect(mockedReadService.setSourceList).not.toHaveBeenCalled();
  });

  it('rolls back and alerts when the update fails', async () => {
    const alertSpy = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    mockedReadService.setSourceList.mockRejectedValue(new Error('boom'));
    renderList([rss({ list: 'reads' })]);
    await screen.findByText('Go Blog');

    fireEvent.press(feedButton('Go Blog'));

    await waitFor(() =>
      expect(alertSpy).toHaveBeenCalledWith('Error', 'Failed to update. Please try again.'),
    );
    expect(isSelected(readsButton('Go Blog'))).toBe(true);
    expect(isSelected(feedButton('Go Blog'))).toBe(false);
  });

  it('offers no toggle for source types the backend cannot route', async () => {
    renderList([
      {
        id: 'soc-1',
        type: 'social',
        title: 'Some Account',
        subscribed_at: '2025-01-01T00:00:00Z',
        list: 'reads',
        social_data: { platform: 'x', handle: 'acct' },
      },
    ]);
    await screen.findByText('Some Account');

    expect(screen.queryByLabelText('Send Some Account to Feed')).toBeNull();
  });
});
