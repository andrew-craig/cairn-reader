import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { Article, UserContentResponse } from '@cairn/shared';
import ReadArticle from './ReadArticle';
import { ReadService } from '../services/read';

// task_3c49 piece 4: archive/delete used to fire immediately with no
// confirmation and surfaced failures only via console.error, invisible to the
// user. Confirmation must gate both actions, and a failure must be visible.

function makeArticle(overrides: Partial<Article> = {}): Article {
  return {
    id: 'article-1',
    url: 'https://example.com',
    title: 'Test Article',
    tags: [],
    isRead: false,
    isFavorite: false,
    list: 'reads' as const,
    addedAt: Date.now(),
    ...overrides,
  };
}

// The reader only reads `.then`s of updateUserContent for side effects; the
// resolved value itself is never inspected by these tests.
const updateResponse = {} as UserContentResponse;

function renderReader() {
  return render(
    <MemoryRouter initialEntries={['/reads/article-1']}>
      <Routes>
        <Route path="/reads/:id" element={<ReadArticle />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('ReadArticle destructive actions', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(ReadService, 'getUserContent').mockResolvedValue(makeArticle());
  });

  it('does not archive when the confirmation is dismissed', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false);
    const updateSpy = vi.spyOn(ReadService, 'updateUserContent').mockResolvedValue(updateResponse);
    renderReader();

    await userEvent.click(await screen.findByRole('button', { name: 'Archive' }));

    expect(window.confirm).toHaveBeenCalled();
    expect(updateSpy).not.toHaveBeenCalledWith('article-1', { status: 'archived' });
  });

  it('archives after the confirmation is accepted', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    const updateSpy = vi.spyOn(ReadService, 'updateUserContent').mockResolvedValue(updateResponse);
    renderReader();

    await userEvent.click(await screen.findByRole('button', { name: 'Archive' }));

    await waitFor(() =>
      expect(updateSpy).toHaveBeenCalledWith('article-1', { status: 'archived' }),
    );
  });

  it('shows a visible error when archiving fails', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.spyOn(ReadService, 'updateUserContent').mockRejectedValue(new Error('network down'));
    renderReader();

    await userEvent.click(await screen.findByRole('button', { name: 'Archive' }));

    await waitFor(() => expect(alertSpy).toHaveBeenCalled());
  });

  it('does not delete when the confirmation is dismissed', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false);
    const deleteSpy = vi.spyOn(ReadService, 'deleteUserContent').mockResolvedValue(undefined);
    renderReader();

    await userEvent.click(await screen.findByRole('button', { name: 'Delete' }));

    expect(window.confirm).toHaveBeenCalled();
    expect(deleteSpy).not.toHaveBeenCalled();
  });

  it('deletes after the confirmation is accepted', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    const deleteSpy = vi.spyOn(ReadService, 'deleteUserContent').mockResolvedValue(undefined);
    renderReader();

    await userEvent.click(await screen.findByRole('button', { name: 'Delete' }));

    await waitFor(() => expect(deleteSpy).toHaveBeenCalledWith('article-1'));
  });

  it('shows a visible error when deleting fails', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.spyOn(ReadService, 'deleteUserContent').mockRejectedValue(new Error('network down'));
    renderReader();

    await userEvent.click(await screen.findByRole('button', { name: 'Delete' }));

    await waitFor(() => expect(alertSpy).toHaveBeenCalled());
  });
});

// task_f4ac: Feed items are skim-only — Save to Reads replaces Archive and no
// reading progress is written.
describe('ReadArticle for Feed items', () => {
  function renderFeedReader() {
    return render(
      <MemoryRouter initialEntries={['/feed/article-1']}>
        <Routes>
          <Route path="/feed/:id" element={<ReadArticle />} />
          <Route path="/feed" element={<p>feed list</p>} />
          <Route path="/reads" element={<p>reads list</p>} />
        </Routes>
      </MemoryRouter>,
    );
  }

  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(ReadService, 'getUserContent').mockResolvedValue(makeArticle({ list: 'feed' }));
  });

  it('offers Save to Reads instead of Archive', async () => {
    renderFeedReader();

    expect(await screen.findByRole('button', { name: 'Save to Reads' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Archive' })).toBeNull();
  });

  it('moves the item to Reads and returns to the feed', async () => {
    const moveSpy = vi.spyOn(ReadService, 'moveToReads').mockResolvedValue(updateResponse);
    renderFeedReader();

    await userEvent.click(await screen.findByRole('button', { name: 'Save to Reads' }));

    await waitFor(() => expect(moveSpy).toHaveBeenCalledWith('article-1'));
    expect(await screen.findByText('feed list')).toBeTruthy();
  });

  it('shows a visible error when saving to Reads fails', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.spyOn(ReadService, 'moveToReads').mockRejectedValue(new Error('network down'));
    renderFeedReader();

    await userEvent.click(await screen.findByRole('button', { name: 'Save to Reads' }));

    await waitFor(() => expect(alertSpy).toHaveBeenCalled());
  });

  it('does not mark a Feed item as reading', async () => {
    const updateSpy = vi.spyOn(ReadService, 'updateUserContent').mockResolvedValue(updateResponse);
    renderFeedReader();
    await screen.findByRole('button', { name: 'Save to Reads' });

    expect(updateSpy).not.toHaveBeenCalled();
  });

  it('still marks a Reads item as reading', async () => {
    vi.spyOn(ReadService, 'getUserContent').mockResolvedValue(makeArticle());
    const updateSpy = vi.spyOn(ReadService, 'updateUserContent').mockResolvedValue(updateResponse);
    render(
      <MemoryRouter initialEntries={['/reads/article-1']}>
        <Routes>
          <Route path="/reads/:id" element={<ReadArticle />} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() =>
      expect(updateSpy).toHaveBeenCalledWith('article-1', { status: 'reading' }),
    );
  });
});

describe('ReadArticle Save to Reads double click', () => {
  it('sends the move once', async () => {
    vi.restoreAllMocks();
    vi.spyOn(ReadService, 'getUserContent').mockResolvedValue(makeArticle({ list: 'feed' }));
    const moveSpy = vi.spyOn(ReadService, 'moveToReads').mockReturnValue(new Promise(() => {}));
    render(
      <MemoryRouter initialEntries={['/feed/article-1']}>
        <Routes>
          <Route path="/feed/:id" element={<ReadArticle />} />
        </Routes>
      </MemoryRouter>,
    );

    const button = await screen.findByRole('button', { name: 'Save to Reads' });
    await userEvent.dblClick(button);

    expect(moveSpy).toHaveBeenCalledTimes(1);
  });
});
