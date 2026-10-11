import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import SourceListToggle from './SourceListToggle';
import { ReadService } from '../services/read';

function renderToggle(list: 'feed' | 'reads' = 'reads') {
  return render(<SourceListToggle type="rss" sourceKey="feed-1" list={list} label="Example" />);
}

describe('SourceListToggle', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('reflects the current list', () => {
    renderToggle('feed');
    expect(screen.getByRole('button', { name: 'Feed' }).getAttribute('aria-pressed')).toBe('true');
    expect(screen.getByRole('button', { name: 'Reads' }).getAttribute('aria-pressed')).toBe('false');
  });

  it('switches optimistically and persists the choice', async () => {
    const spy = vi
      .spyOn(ReadService, 'setSourceList')
      .mockResolvedValue({ type: 'rss', key: 'feed-1', list: 'feed' });
    renderToggle('reads');

    await userEvent.click(screen.getByRole('button', { name: 'Feed' }));

    expect(screen.getByRole('button', { name: 'Feed' }).getAttribute('aria-pressed')).toBe('true');
    expect(spy).toHaveBeenCalledWith('rss', 'feed-1', 'feed');
  });

  it('does nothing when the current list is chosen again', async () => {
    const spy = vi.spyOn(ReadService, 'setSourceList');
    renderToggle('reads');

    await userEvent.click(screen.getByRole('button', { name: 'Reads' }));

    expect(spy).not.toHaveBeenCalled();
  });

  it('rolls back and shows an error when the update fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.spyOn(ReadService, 'setSourceList').mockRejectedValue(new Error('nope'));
    renderToggle('reads');

    await userEvent.click(screen.getByRole('button', { name: 'Feed' }));

    await waitFor(() => expect(screen.getByRole('alert')).toBeTruthy());
    expect(screen.getByRole('button', { name: 'Reads' }).getAttribute('aria-pressed')).toBe('true');
  });
});
