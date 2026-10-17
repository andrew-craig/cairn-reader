import React from 'react';
import { Alert } from 'react-native';
import { render, screen, fireEvent, waitFor } from '@testing-library/react-native';
import type { UnifiedSubscription } from '@cairn/shared';
import { RssScreen } from './RssScreen';
import { ReadService } from '../services/read';

jest.mock('../services/read', () => ({
  ReadService: {
    listAllSubscriptions: jest.fn(),
    detectURL: jest.fn(),
    addURL: jest.fn(),
  },
}));

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ canGoBack: () => true, goBack: jest.fn() }),
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  useFocusEffect: (cb: () => void) => require('react').useEffect(() => cb(), [cb]),
}));

const mockedReadService = ReadService as jest.Mocked<typeof ReadService>;

const goBlog: UnifiedSubscription = {
  id: 'sub-rss',
  type: 'rss',
  title: 'Go Blog',
  subscribed_at: '2025-01-01T00:00:00Z',
  list: 'feed',
  rss_data: { feed_id: 'feed-1', feed_url: 'https://go.dev/feed' },
};

// feature_bd13: the RSS page can add a feed itself, with the Feed/Reads choice.
describe('RssScreen Add feed', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(Alert, 'alert').mockImplementation(() => {});
  });

  it('subscribes to the chosen list and shows the new feed without leaving the page', async () => {
    mockedReadService.listAllSubscriptions
      .mockResolvedValueOnce({ subscriptions: [], total_count: 0, failed_sources: [] })
      .mockResolvedValue({ subscriptions: [goBlog], total_count: 1, failed_sources: [] });
    mockedReadService.detectURL.mockResolvedValue({ url: 'https://go.dev/feed', type: 'feed', title: 'Go Blog' });
    mockedReadService.addURL.mockResolvedValue({
      type: 'feed',
      subscription: { title: 'Go Blog', list: 'feed' },
    } as never);

    render(<RssScreen />);
    expect(await screen.findByText('No RSS feeds yet')).toBeTruthy();

    fireEvent.press(screen.getByLabelText('Add feed'));
    fireEvent.changeText(screen.getByPlaceholderText('Feed or website URL'), 'https://go.dev/feed');
    await screen.findByText('Add Feed');
    fireEvent.press(screen.getByRole('radio', { name: /^Feed:/ }));
    fireEvent.press(screen.getByText('Add Feed'));

    await waitFor(() =>
      expect(mockedReadService.addURL).toHaveBeenCalledWith(
        expect.objectContaining({ url: 'https://go.dev/feed', type: 'feed', list: 'feed' }),
      ),
    );
    expect(await screen.findByText('Go Blog')).toBeTruthy();
    expect(mockedReadService.listAllSubscriptions).toHaveBeenCalledTimes(2);
  });
});
