import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
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
});
