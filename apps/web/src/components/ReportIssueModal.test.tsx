import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ReportIssueModal from './ReportIssueModal';
import { ReadService, type ArticleFeedback } from '../services/read';

const existing: ArticleFeedback = {
  content_id: 'c-1',
  categories: ['images', 'title'],
  comment: 'Hero image is broken',
  status: 'open',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

describe('ReportIssueModal', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(ReadService, 'getArticleFeedback').mockResolvedValue(null);
  });

  it('disables Submit until a reason is selected', async () => {
    render(<ReportIssueModal contentId="c-1" onClose={vi.fn()} />);
    const submit = screen.getByRole('button', { name: 'Submit' }) as HTMLButtonElement;
    expect(submit.disabled).toBe(true);
    await userEvent.click(screen.getByLabelText('Images are missing or broken'));
    expect(submit.disabled).toBe(false);
    await userEvent.click(screen.getByLabelText('Images are missing or broken'));
    expect(submit.disabled).toBe(true);
  });

  it('prefills reasons and details from an existing report', async () => {
    vi.spyOn(ReadService, 'getArticleFeedback').mockResolvedValue(existing);
    render(<ReportIssueModal contentId="c-1" onClose={vi.fn()} />);
    await waitFor(() =>
      expect((screen.getByLabelText('Images are missing or broken') as HTMLInputElement).checked).toBe(true),
    );
    expect((screen.getByLabelText('Title is wrong') as HTMLInputElement).checked).toBe(true);
    expect((screen.getByLabelText('Author is wrong') as HTMLInputElement).checked).toBe(false);
    expect((screen.getByLabelText('Details (optional)') as HTMLTextAreaElement).value).toBe(
      'Hero image is broken',
    );
  });

  it('submits the selected categories and comment, then confirms and closes', async () => {
    const submit = vi.spyOn(ReadService, 'submitArticleFeedback').mockResolvedValue(existing);
    const onClose = vi.fn();
    render(<ReportIssueModal contentId="c-1" onClose={onClose} />);
    await userEvent.click(screen.getByLabelText('Title is wrong'));
    await userEvent.click(screen.getByLabelText('Links are missing or broken'));
    await userEvent.type(screen.getByLabelText('Details (optional)'), 'Wrong headline');
    await userEvent.click(screen.getByRole('button', { name: 'Submit' }));
    expect(submit).toHaveBeenCalledWith('c-1', {
      categories: ['links', 'title'],
      comment: 'Wrong headline',
    });
    expect(await screen.findByText('Thanks — your report was sent.')).toBeTruthy();
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1), { timeout: 3000 });
  });

  it('shows an inline error on failure and keeps the selection', async () => {
    vi.spyOn(ReadService, 'submitArticleFeedback').mockRejectedValue(new Error('boom'));
    vi.spyOn(console, 'error').mockImplementation(() => {});
    const onClose = vi.fn();
    render(<ReportIssueModal contentId="c-1" onClose={onClose} />);
    await userEvent.click(screen.getByLabelText('Title is wrong'));
    await userEvent.click(screen.getByRole('button', { name: 'Submit' }));
    expect(await screen.findByText(/Couldn't send your report/)).toBeTruthy();
    expect((screen.getByLabelText('Title is wrong') as HTMLInputElement).checked).toBe(true);
    expect((screen.getByRole('button', { name: 'Submit' }) as HTMLButtonElement).disabled).toBe(false);
    expect(onClose).not.toHaveBeenCalled();
  });

  it('closes on Escape, Cancel and the backdrop', async () => {
    const onClose = vi.fn();
    render(<ReportIssueModal contentId="c-1" onClose={onClose} />);
    await userEvent.keyboard('{Escape}');
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    await userEvent.click(document.querySelector('.add-link-overlay__backdrop') as HTMLElement);
    expect(onClose).toHaveBeenCalledTimes(3);
  });
});
