import React from 'react';
import { render, fireEvent, screen } from '@testing-library/react-native';
import { BottomActionMenu } from './BottomActionMenu';

describe('BottomActionMenu', () => {
  it('labels every action for screen readers', () => {
    render(
      <BottomActionMenu
        actions={[
          { icon: 'return', label: 'Back', onPress: () => {} },
          { icon: 'next-article', label: 'Next', onPress: () => {} },
          { icon: 'bookmark', label: 'Favorite', onPress: () => {} },
          { icon: 'archive', label: 'Archive', onPress: () => {} },
        ]}
      />
    );

    expect(screen.getByLabelText('Back')).toBeTruthy();
    expect(screen.getByLabelText('Next')).toBeTruthy();
    expect(screen.getByLabelText('Favorite')).toBeTruthy();
    expect(screen.getByLabelText('Archive')).toBeTruthy();
  });

  it('calls the action onPress when tapped', () => {
    const onPress = jest.fn();
    render(<BottomActionMenu actions={[{ icon: 'archive', label: 'Archive', onPress }]} />);
    fireEvent.press(screen.getByLabelText('Archive'));
    expect(onPress).toHaveBeenCalledTimes(1);
  });
});
