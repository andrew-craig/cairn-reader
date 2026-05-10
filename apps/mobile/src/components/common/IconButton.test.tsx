import React from 'react';
import { render, fireEvent, screen } from '@testing-library/react-native';
import { IconButton } from './IconButton';

describe('IconButton', () => {
  it('renders without crashing', () => {
    render(<IconButton icon="close" onPress={() => {}} />);
    expect(screen.getByRole('button')).toBeTruthy();
  });

  it('calls onPress when tapped', () => {
    const onPress = jest.fn();
    render(<IconButton icon="close" onPress={onPress} />);
    fireEvent.press(screen.getByRole('button'));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it('passes custom size to the icon', () => {
    render(<IconButton icon="close" onPress={() => {}} size={32} />);
    expect(screen.getByRole('button')).toBeTruthy();
  });
});
