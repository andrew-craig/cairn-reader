import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import type { UserContentsListResponse } from '@cairn/shared';
import ArticleList from './ArticleList';
import { ReadService } from '../services/read';

const emptyPage: UserContentsListResponse = {
  contents: [],
  total_count: 0,
  limit: 20,
  cursor: '',
  has_more: false,
};

function renderList(list: 'feed' | 'reads') {
  return render(
    <MemoryRouter>
      <ArticleList list={list} />
    </MemoryRouter>,
  );
}

describe('ArticleList', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.stubGlobal(
      'IntersectionObserver',
      class {
        observe() {}
        disconnect() {}
      },
    );
  });

  it.each(['feed', 'reads'] as const)('fetches only the %s list', async (list) => {
    const spy = vi.spyOn(ReadService, 'listUserContents').mockResolvedValue(emptyPage);
    renderList(list);

    await waitFor(() => expect(spy).toHaveBeenCalledWith(expect.objectContaining({ list })));
  });

  it('offers Add link on Reads but not on Feed', async () => {
    vi.spyOn(ReadService, 'listUserContents').mockResolvedValue(emptyPage);

    const reads = renderList('reads');
    await screen.findByText('No saved articles yet.');
    expect(screen.getByRole('button', { name: 'Add link' })).toBeTruthy();
    reads.unmount();

    renderList('feed');
    await screen.findByText(/Your feed is empty/);
    expect(screen.queryByRole('button', { name: 'Add link' })).toBeNull();
  });

  it('opens Add link from the plus icon', async () => {
    vi.spyOn(ReadService, 'listUserContents').mockResolvedValue(emptyPage);
    renderList('reads');
    await screen.findByText('No saved articles yet.');

    fireEvent.click(screen.getByRole('button', { name: 'Add link' }));
    expect(screen.getByRole('dialog')).toBeTruthy();
  });

  it('toggles an inline search field that shows results in place of the list', async () => {
    vi.spyOn(ReadService, 'listUserContents').mockResolvedValue(emptyPage);
    const search = vi.spyOn(ReadService, 'searchUserContents').mockResolvedValue({
      ...emptyPage,
      contents: [],
    });
    renderList('reads');
    await screen.findByText('No saved articles yet.');

    const toggle = screen.getByRole('button', { name: 'Search' });
    expect(toggle.getAttribute('aria-expanded')).toBe('false');

    fireEvent.click(toggle);
    const input = screen.getByRole('searchbox', { name: 'Search your reads' });
    expect(toggle.getAttribute('aria-expanded')).toBe('true');
    expect(document.activeElement).toBe(input);

    fireEvent.change(input, { target: { value: 'rust' } });
    await screen.findByText('No results.');
    expect(search).toHaveBeenCalledWith({ q: 'rust', list: 'reads' });
    expect(screen.queryByText('No saved articles yet.')).toBeNull();

    fireEvent.keyDown(input, { key: 'Escape' });
    expect(toggle.getAttribute('aria-expanded')).toBe('false');
    expect(document.activeElement).toBe(toggle);
    expect(screen.getByText('No saved articles yet.')).toBeTruthy();
  });

  it('closes the search field from its close button', async () => {
    vi.spyOn(ReadService, 'listUserContents').mockResolvedValue(emptyPage);
    renderList('reads');
    await screen.findByText('No saved articles yet.');

    const toggle = screen.getByRole('button', { name: 'Search' });
    fireEvent.click(toggle);
    fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'rust' } });

    fireEvent.click(screen.getByRole('button', { name: 'Close search' }));
    expect(toggle.getAttribute('aria-expanded')).toBe('false');
    expect(document.activeElement).toBe(toggle);
    expect((screen.getByRole('searchbox', { hidden: true }) as HTMLInputElement).value).toBe('');
  });
});
