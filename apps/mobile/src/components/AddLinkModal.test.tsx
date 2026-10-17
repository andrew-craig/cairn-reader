import React from 'react';
import { Alert, Modal } from 'react-native';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react-native';
import { AddLinkModal } from './AddLinkModal';
import { ReadService } from '../services/read';

// task_7c06: dismissal must not be gated on `detecting` — detection is not
// user-initiated (fires on every URL change), so an in-flight detection must
// never trap the modal.

describe('AddLinkModal dismissal during URL detection', () => {
  afterEach(() => jest.restoreAllMocks());

  it('closes on request while a detection is in flight', async () => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    // Detection never resolves -> modal stays in the `detecting` state.
    jest.spyOn(ReadService, 'detectURL').mockReturnValue(new Promise(() => {}));
    const onClose = jest.fn();

    render(<AddLinkModal visible onClose={onClose} />);
    fireEvent.changeText(screen.getByPlaceholderText('Add link'), 'https://example.com');

    await waitFor(() => expect(ReadService.detectURL).toHaveBeenCalled());

    fireEvent(screen.UNSAFE_getByType(Modal), 'requestClose');
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('still blocks dismissal while a real submit is in flight', async () => {
    jest.spyOn(ReadService, 'detectURL').mockResolvedValue({
      url: 'https://example.com',
      type: 'page',
      title: null,
    });
    jest.spyOn(ReadService, 'addURL').mockReturnValue(new Promise(() => {}));
    const onClose = jest.fn();

    render(<AddLinkModal visible onClose={onClose} />);
    fireEvent.changeText(screen.getByPlaceholderText('Add link'), 'https://example.com');

    const addButton = await screen.findByText('Add');
    fireEvent.press(addButton);

    await waitFor(() => expect(ReadService.addURL).toHaveBeenCalled());

    fireEvent(screen.UNSAFE_getByType(Modal), 'requestClose');
    expect(onClose).not.toHaveBeenCalled();
  });
});

// task_7c06 follow-up: dismissal must not be gated on `discovering` either,
// and a late discoverFeed resolution after close must not fire the
// 'Multiple feeds found' Alert (or otherwise touch state).

describe('AddLinkModal dismissal during Find-feed', () => {
  afterEach(() => jest.restoreAllMocks());

  it('does not fire the Multiple feeds Alert when discoverFeed resolves after close', async () => {
    const alertSpy = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    jest.spyOn(ReadService, 'detectURL').mockResolvedValue({
      url: 'https://example.com',
      type: 'page',
      title: null,
    });
    let resolveDiscover: (value: { feeds: { url: string; title: string }[] }) => void;
    jest.spyOn(ReadService, 'discoverFeed').mockReturnValue(
      new Promise((resolve) => {
        resolveDiscover = resolve;
      }),
    );
    const onClose = jest.fn();

    render(<AddLinkModal visible onClose={onClose} />);
    fireEvent.changeText(screen.getByPlaceholderText('Add link'), 'https://example.com');

    const findFeedButton = await screen.findByText('Find feed');
    fireEvent.press(findFeedButton);

    await waitFor(() => expect(ReadService.discoverFeed).toHaveBeenCalled());

    fireEvent(screen.UNSAFE_getByType(Modal), 'requestClose');
    expect(onClose).toHaveBeenCalledTimes(1);

    resolveDiscover!({
      feeds: [
        { url: 'https://example.com/feed1', title: 'Feed 1' },
        { url: 'https://example.com/feed2', title: 'Feed 2' },
      ],
    });
    // Flush the resolved promise; the Alert must not fire post-close.
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(alertSpy).not.toHaveBeenCalled();
  });
});

// Epic 6e4d: subscribing to a feed asks where its new items should land.

describe('AddLinkModal Feed/Reads choice for feeds', () => {
  afterEach(() => jest.restoreAllMocks());

  const feedDetection = { url: 'https://example.com/rss', type: 'feed' as const, title: 'Example' };
  const feedResponse = (list: 'feed' | 'reads') =>
    ({
      type: 'feed',
      feed_id: 'f1',
      subscription: {
        id: 's1',
        user_id: 'u',
        feed_id: 'f1',
        feed_url: 'https://example.com/rss',
        title: 'Example',
        list,
        subscribed_at: '2025-01-01T00:00:00Z',
      },
    }) as never;

  it('defaults a new feed subscription to Reads', async () => {
    jest.spyOn(ReadService, 'detectURL').mockResolvedValue(feedDetection);
    jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    const addSpy = jest.spyOn(ReadService, 'addURL').mockResolvedValue(feedResponse('reads'));

    render(<AddLinkModal visible onClose={jest.fn()} />);
    fireEvent.changeText(screen.getByPlaceholderText('Add link'), 'https://example.com/rss');
    fireEvent.press(await screen.findByText('Add Feed'));

    await waitFor(() => expect(addSpy).toHaveBeenCalled());
    expect(addSpy).toHaveBeenCalledWith(expect.objectContaining({ type: 'feed', list: 'reads' }));
  });

  it('subscribes to Feed when the user picks Feed, and says where items will land', async () => {
    jest.spyOn(ReadService, 'detectURL').mockResolvedValue(feedDetection);
    const alertSpy = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    const addSpy = jest.spyOn(ReadService, 'addURL').mockResolvedValue(feedResponse('feed'));

    render(<AddLinkModal visible onClose={jest.fn()} />);
    fireEvent.changeText(screen.getByPlaceholderText('Add link'), 'https://example.com/rss');
    await screen.findByText('Add Feed');
    fireEvent.press(screen.getByRole('radio', { name: /^Feed:/ }));
    fireEvent.press(screen.getByText('Add Feed'));

    await waitFor(() => expect(addSpy).toHaveBeenCalled());
    expect(addSpy).toHaveBeenCalledWith(expect.objectContaining({ type: 'feed', list: 'feed' }));
    await waitFor(() =>
      expect(alertSpy).toHaveBeenCalledWith('Success', 'Subscribed to Example. New items will appear in Feed.', [
        { text: 'OK' },
      ]),
    );
  });

  it('drops a stale feed detection when the URL changes, and ignores submit presses until the new URL is detected', async () => {
    let resolvePage: (value: { url: string; type: 'page'; title: null }) => void;
    jest.spyOn(ReadService, 'detectURL').mockImplementation((url: string) =>
      url.includes('/rss')
        ? Promise.resolve(feedDetection)
        : new Promise((resolve) => {
            resolvePage = resolve as typeof resolvePage;
          }),
    );
    jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    const addSpy = jest.spyOn(ReadService, 'addURL').mockResolvedValue({ type: 'page' } as never);

    render(<AddLinkModal visible onClose={jest.fn()} />);
    const input = screen.getByPlaceholderText('Add link');
    fireEvent.changeText(input, 'https://example.com/rss');
    await screen.findByText('Add Feed');

    // Edit to a different URL: the feed result and its Feed/Reads choice go away.
    fireEvent.changeText(input, 'https://example.com/post');
    await waitFor(() => expect(screen.queryByText('Add Feed')).toBeNull());
    expect(screen.queryByRole('radio', { name: /^Feed:/ })).toBeNull();

    // Submitting now would drop type/list for lack of a detection, so the press is ignored.
    fireEvent(input, 'submitEditing');
    expect(addSpy).not.toHaveBeenCalled();

    // Once the new URL is detected (as a page), submit goes out without a list.
    await act(async () => {
      resolvePage!({ url: 'https://example.com/post', type: 'page', title: null });
    });
    fireEvent(input, 'submitEditing');
    await waitFor(() => expect(addSpy).toHaveBeenCalled());
    expect(addSpy.mock.calls[0][0]).toMatchObject({ type: 'page' });
    expect(addSpy.mock.calls[0][0]).not.toHaveProperty('list');
  });

  it('does not stay stuck on Detecting when the URL becomes invalid mid-detection', async () => {
    jest.spyOn(ReadService, 'detectURL').mockReturnValue(new Promise(() => {}));
    const addSpy = jest.spyOn(ReadService, 'addURL');

    render(<AddLinkModal visible onClose={jest.fn()} />);
    const input = screen.getByPlaceholderText('Add link');
    fireEvent.changeText(input, 'https://example.com/post');
    await waitFor(() => expect(ReadService.detectURL).toHaveBeenCalled());

    // The URL becomes invalid while that detection is still in flight.
    fireEvent.changeText(input, 'not a url');
    fireEvent(input, 'submitEditing');

    expect(await screen.findByText('Please enter a valid URL')).toBeTruthy();
    expect(addSpy).not.toHaveBeenCalled();
  });

  it('does not offer the choice for a page, and sends no list', async () => {
    jest.spyOn(ReadService, 'detectURL').mockResolvedValue({
      url: 'https://example.com/post',
      type: 'page',
      title: null,
    });
    jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    const addSpy = jest.spyOn(ReadService, 'addURL').mockResolvedValue({ type: 'page' } as never);

    render(<AddLinkModal visible onClose={jest.fn()} />);
    fireEvent.changeText(screen.getByPlaceholderText('Add link'), 'https://example.com/post');
    fireEvent.press(await screen.findByText('Add'));

    expect(screen.queryByRole('radio', { name: /^Feed:/ })).toBeNull();
    await waitFor(() => expect(addSpy).toHaveBeenCalled());
    expect(addSpy.mock.calls[0][0]).not.toHaveProperty('list');
  });
});

// feature_bd13 / task_9188: the RSS page opens the same modal in feed mode.

describe('AddLinkModal feed mode', () => {
  afterEach(() => jest.restoreAllMocks());

  it('explains both destinations when a feed is detected', async () => {
    jest.spyOn(ReadService, 'detectURL').mockResolvedValue({
      url: 'https://example.com/rss',
      type: 'feed',
      title: 'Example',
    });

    render(<AddLinkModal mode="feed" visible onClose={jest.fn()} />);
    fireEvent.changeText(screen.getByPlaceholderText('Feed or website URL'), 'https://example.com/rss');
    await screen.findByText('Add Feed');

    expect(screen.getByText('Where should new items go?')).toBeTruthy();
    expect(screen.getByRole('radio', { name: /^Feed:.*deleted after 30 days/ })).toBeTruthy();
    expect(screen.getByRole('radio', { name: /^Reads:/, checked: true })).toBeTruthy();
  });

  it('never saves a page as an article: Find feed is the only action, and return runs it', async () => {
    jest.spyOn(ReadService, 'detectURL').mockResolvedValue({
      url: 'https://example.com',
      type: 'page',
      title: null,
    });
    const discoverSpy = jest
      .spyOn(ReadService, 'discoverFeed')
      .mockResolvedValue({ feeds: [{ url: 'https://example.com/rss', title: 'Example' }] } as never);
    const addSpy = jest.spyOn(ReadService, 'addURL');

    render(<AddLinkModal mode="feed" visible onClose={jest.fn()} />);
    const input = screen.getByPlaceholderText('Feed or website URL');
    fireEvent.changeText(input, 'https://example.com');
    await screen.findByText('Find feed');

    expect(screen.queryByText('Add')).toBeNull();
    fireEvent(input, 'submitEditing');

    await waitFor(() => expect(discoverSpy).toHaveBeenCalledWith('https://example.com'));
    expect(addSpy).not.toHaveBeenCalled();
  });

  it('runs Find feed on return while detection is still in flight', async () => {
    jest.spyOn(ReadService, 'detectURL').mockReturnValue(new Promise(() => {}));
    const discoverSpy = jest
      .spyOn(ReadService, 'discoverFeed')
      .mockResolvedValue({ feeds: [{ url: 'https://example.com/rss', title: 'Example' }] } as never);

    render(<AddLinkModal mode="feed" visible onClose={jest.fn()} />);
    const input = screen.getByPlaceholderText('Feed or website URL');
    fireEvent.changeText(input, 'https://example.com');
    await waitFor(() => expect(ReadService.detectURL).toHaveBeenCalled());

    expect(screen.queryByText('Add')).toBeNull();
    fireEvent(input, 'submitEditing');

    await waitFor(() => expect(discoverSpy).toHaveBeenCalledWith('https://example.com'));
  });
});
