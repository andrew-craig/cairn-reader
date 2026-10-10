import React from 'react';
import { Alert, Modal } from 'react-native';
import { render, screen, fireEvent, waitFor } from '@testing-library/react-native';
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
    fireEvent.press(screen.getByLabelText('Send this feed to Feed'));
    fireEvent.press(screen.getByText('Add Feed'));

    await waitFor(() => expect(addSpy).toHaveBeenCalled());
    expect(addSpy).toHaveBeenCalledWith(expect.objectContaining({ type: 'feed', list: 'feed' }));
    await waitFor(() =>
      expect(alertSpy).toHaveBeenCalledWith('Success', 'Subscribed to Example. New items will appear in Feed.', [
        { text: 'OK' },
      ]),
    );
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

    expect(screen.queryByLabelText('Send this feed to Feed')).toBeNull();
    await waitFor(() => expect(addSpy).toHaveBeenCalled());
    expect(addSpy.mock.calls[0][0]).not.toHaveProperty('list');
  });
});
