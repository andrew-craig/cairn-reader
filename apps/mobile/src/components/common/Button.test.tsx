import React from 'react';
import { ActivityIndicator } from 'react-native';
import { render, fireEvent, screen } from '@testing-library/react-native';
import { Button } from './Button';

describe('Button', () => {
  it('renders the title', () => {
    render(<Button title="Save" onPress={() => {}} />);
    expect(screen.getByText('Save')).toBeTruthy();
  });

  it('calls onPress when tapped', () => {
    const onPress = jest.fn();
    render(<Button title="Save" onPress={onPress} />);
    fireEvent.press(screen.getByText('Save'));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it('disables the touchable when disabled prop is set', () => {
    render(<Button title="Save" onPress={() => {}} disabled />);
    expect(screen.getByRole('button', { disabled: true })).toBeTruthy();
  });

  it('hides title and shows ActivityIndicator when loading', () => {
    render(<Button title="Save" onPress={() => {}} loading />);
    expect(screen.queryByText('Save')).toBeNull();
    expect(screen.UNSAFE_getByType(ActivityIndicator)).toBeTruthy();
  });

  it('disables the touchable when loading', () => {
    render(<Button title="Save" onPress={() => {}} loading />);
    expect(screen.getByRole('button', { disabled: true })).toBeTruthy();
  });

  it('renders secondary and outline variants without crashing', () => {
    const { rerender } = render(<Button title="Save" onPress={() => {}} variant="secondary" />);
    expect(screen.getByText('Save')).toBeTruthy();
    rerender(<Button title="Save" onPress={() => {}} variant="outline" />);
    expect(screen.getByText('Save')).toBeTruthy();
  });
});
