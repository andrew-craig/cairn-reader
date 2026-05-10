import React from 'react';
import { Text } from 'react-native';
import { render, fireEvent, screen } from '@testing-library/react-native';
import { ScreenHeader } from './ScreenHeader';

describe('ScreenHeader', () => {
  it('renders the title', () => {
    render(<ScreenHeader title="Settings" />);
    expect(screen.getByText('Settings')).toBeTruthy();
  });

  it('does not render a back button when onBack is not provided', () => {
    render(<ScreenHeader title="Settings" />);
    expect(screen.queryByRole('button')).toBeNull();
  });

  it('calls onBack when the back button is pressed', () => {
    const onBack = jest.fn();
    render(<ScreenHeader title="Settings" onBack={onBack} />);
    fireEvent.press(screen.getByRole('button'));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it('renders right actions when provided', () => {
    render(
      <ScreenHeader
        title="Settings"
        rightActions={<Text testID="right-action">Edit</Text>}
      />
    );
    expect(screen.getByTestId('right-action')).toBeTruthy();
  });
});
