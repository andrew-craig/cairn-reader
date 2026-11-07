import React from 'react';
import { Alert } from 'react-native';
import { render, screen, fireEvent, waitFor } from '@testing-library/react-native';
import { ReportIssueModal } from './ReportIssueModal';
import { ReadService } from '../services/read';

const IMAGES = 'Images are missing or broken';
const TITLE = 'Title is wrong';

describe('ReportIssueModal', () => {
  beforeEach(() => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(ReadService, 'getExtractionFeedback').mockResolvedValue(null);
  });
  afterEach(() => jest.restoreAllMocks());

  const renderModal = (onClose = jest.fn()) => {
    render(<ReportIssueModal visible articleId="c-1" onClose={onClose} />);
    return onClose;
  };

  const submitButton = () => screen.getByText('Submit');

  it('disables Submit until a reason is selected', async () => {
    renderModal();
    await waitFor(() => expect(ReadService.getExtractionFeedback).toHaveBeenCalledWith('c-1'));

    expect(screen.getByRole('button', { name: 'Submit' })).toBeDisabled();
    fireEvent.press(screen.getByRole('checkbox', { name: IMAGES }));
    expect(screen.getByRole('checkbox', { name: IMAGES })).toBeChecked();
    expect(screen.getByRole('button', { name: 'Submit' })).toBeEnabled();

    fireEvent.press(screen.getByRole('checkbox', { name: IMAGES }));
    expect(screen.getByRole('button', { name: 'Submit' })).toBeDisabled();
  });

  it('prefills from an existing report', async () => {
    jest.spyOn(ReadService, 'getExtractionFeedback').mockResolvedValue({
      content_id: 'c-1',
      categories: ['title'],
      comment: 'earlier note',
      status: 'open',
      created_at: '',
      updated_at: '',
    });
    renderModal();

    await waitFor(() => expect(screen.getByRole('checkbox', { name: TITLE })).toBeChecked());
    expect(screen.getByDisplayValue('earlier note')).toBeTruthy();
    expect(screen.getByRole('checkbox', { name: IMAGES })).not.toBeChecked();
  });

  it('submits categories and comment, confirms and closes', async () => {
    const alertSpy = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    const submit = jest.spyOn(ReadService, 'submitExtractionFeedback').mockResolvedValue({
      content_id: 'c-1',
      categories: [],
      comment: null,
      status: 'open',
      created_at: '',
      updated_at: '',
    });
    const onClose = renderModal();
    await waitFor(() => expect(ReadService.getExtractionFeedback).toHaveBeenCalled());

    // Tapped out of order: categories are sent in list order.
    fireEvent.press(screen.getByRole('checkbox', { name: TITLE }));
    fireEvent.press(screen.getByRole('checkbox', { name: IMAGES }));
    fireEvent.changeText(screen.getByPlaceholderText('Details (optional)'), 'see top');
    fireEvent.press(submitButton());

    await waitFor(() =>
      expect(submit).toHaveBeenCalledWith('c-1', { categories: ['images', 'title'], comment: 'see top' }),
    );
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(alertSpy).toHaveBeenCalledWith('Report sent', 'Thanks — your report was sent.', expect.anything());
  });

  it('limits the details to 2000 characters', async () => {
    renderModal();
    await waitFor(() => expect(ReadService.getExtractionFeedback).toHaveBeenCalled());

    expect(screen.getByPlaceholderText('Details (optional)').props.maxLength).toBe(2000);
  });

  it('shows an inline error on failure and keeps the selection', async () => {
    jest.spyOn(ReadService, 'submitExtractionFeedback').mockRejectedValue(new Error('Network down'));
    const onClose = renderModal();
    await waitFor(() => expect(ReadService.getExtractionFeedback).toHaveBeenCalled());

    fireEvent.press(screen.getByRole('checkbox', { name: IMAGES }));
    fireEvent.press(submitButton());

    expect(await screen.findByText('Network down')).toBeTruthy();
    expect(screen.getByRole('checkbox', { name: IMAGES })).toBeChecked();
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: 'Submit' })).toBeEnabled();
  });
});
