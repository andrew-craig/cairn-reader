import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import AddLinkModal from './AddLinkModal';
import { ReadService } from '../services/read';

// task_7c06: gating dismissal on `detecting` traps the user — detection is not
// user-initiated (fires on a 400ms debounce on every URL change). Escape, the
// Cancel button and the backdrop must all dismiss while a detection is pending.

describe('AddLinkModal dismissal during URL detection', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    // Detection never resolves -> the modal stays in the `detecting` state.
    vi.spyOn(ReadService, 'detectURL').mockReturnValue(new Promise(() => {}));
  });

  async function openAndStartDetecting() {
    const onClose = vi.fn();
    render(<AddLinkModal onClose={onClose} />);
    const input = screen.getByLabelText('URL to add');
    await userEvent.type(input, 'https://example.com');
    await waitFor(() => expect(ReadService.detectURL).toHaveBeenCalled());
    await screen.findByText('Detecting…');
    return onClose;
  }

  it('Escape dismisses while detecting', async () => {
    const onClose = await openAndStartDetecting();
    await userEvent.keyboard('{Escape}');
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('the Cancel button stays enabled and dismisses while detecting', async () => {
    const onClose = await openAndStartDetecting();
    const cancel = screen.getByRole('button', { name: 'Cancel' });
    expect((cancel as HTMLButtonElement).disabled).toBe(false);
    await userEvent.click(cancel);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('the backdrop dismisses while detecting', async () => {
    const onClose = await openAndStartDetecting();
    const backdrop = document.querySelector('.add-link-overlay__backdrop') as HTMLElement;
    await userEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('a real submit in flight still blocks dismissal', async () => {
    vi.spyOn(ReadService, 'addURL').mockReturnValue(new Promise(() => {}));
    const onClose = vi.fn();
    render(<AddLinkModal onClose={onClose} />);
    await userEvent.type(screen.getByLabelText('URL to add'), 'https://example.com');
    await userEvent.click(screen.getByRole('button', { name: /^Add/ }));
    await screen.findByText('Adding…');
    await userEvent.keyboard('{Escape}');
    expect(onClose).not.toHaveBeenCalled();
  });
});
