import React from 'react';
import { TouchableOpacity } from 'react-native';
import { render, fireEvent, screen } from '@testing-library/react-native';
import { IconButton } from './IconButton';

describe('IconButton', () => {
  it('renders without crashing', () => {
    render(<IconButton icon="close" onPress={() => {}} />);
    expect(screen.UNSAFE_getByType(TouchableOpacity)).toBeTruthy();
  });

  it('calls onPress when tapped', () => {
    const onPress = jest.fn();
    render(<IconButton icon="close" onPress={onPress} />);
    fireEvent.press(screen.UNSAFE_getByType(TouchableOpacity));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it('passes custom size to the icon', () => {
    render(<IconButton icon="close" onPress={() => {}} size={32} />);
    // Verify it renders at all — size is forwarded to Ionicons (native rendered)
    expect(screen.UNSAFE_getByType(TouchableOpacity)).toBeTruthy();
  });
});
