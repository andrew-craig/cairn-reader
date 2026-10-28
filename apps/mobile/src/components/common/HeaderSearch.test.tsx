import React from 'react';
import { render, fireEvent, screen } from '@testing-library/react-native';
import { HeaderSearch } from './HeaderSearch';

function renderSearch(open: boolean) {
  const props = {
    open,
    query: open ? 'rust' : '',
    onChangeQuery: jest.fn(),
    onOpen: jest.fn(),
    onClose: jest.fn(),
    placeholder: 'Search your reads',
    leftEdge: 0,
  };
  render(<HeaderSearch {...props} />);
  return props;
}

describe('HeaderSearch', () => {
  it('opens from the search button', () => {
    const props = renderSearch(false);
    const toggle = screen.getByLabelText('Search');
    expect(toggle.props.accessibilityState).toEqual(expect.objectContaining({ expanded: false }));
    fireEvent.press(toggle);
    expect(props.onOpen).toHaveBeenCalledTimes(1);
  });

  it('reports typing and closes from the close button', () => {
    const props = renderSearch(true);
    fireEvent.changeText(screen.getByLabelText('Search your reads'), 'rust lang');
    expect(props.onChangeQuery).toHaveBeenCalledWith('rust lang');
    fireEvent.press(screen.getByLabelText('Close search'));
    expect(props.onClose).toHaveBeenCalledTimes(1);
  });

  it('disables the search button while open', () => {
    const props = renderSearch(true);
    fireEvent.press(screen.getByLabelText('Search', { includeHiddenElements: true }));
    expect(props.onOpen).not.toHaveBeenCalled();
  });
});
