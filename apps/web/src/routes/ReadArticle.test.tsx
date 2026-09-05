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
    addedAt: Date.now(),
    ...overrides,
  };
}

// The reader only reads `.then`s of updateUserContent for side effects; the
// resolved value itself is never inspected by these tests.
const updateResponse = {} as UserContentResponse;

function renderReader() {
  return render(
    <MemoryRouter initialEntries={['/read/article-1']}>
      <Routes>
        <Route path="/read/:id" element={<ReadArticle />} />
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
