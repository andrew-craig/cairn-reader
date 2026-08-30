import React from 'react';
import { Text } from 'react-native';
import { render, screen, fireEvent } from '@testing-library/react-native';
import { ErrorBoundary } from './ErrorBoundary';

function Boom(): never {
  throw new Error('render exploded');
}

describe('ErrorBoundary', () => {
  it('shows a recoverable fallback instead of crashing on a render error', () => {
    jest.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <ErrorBoundary>
        <Boom />
      </ErrorBoundary>,
    );

    expect(screen.getByText('Something went wrong')).toBeTruthy();
    expect(screen.getByLabelText('Try again')).toBeTruthy();

    jest.restoreAllMocks();
  });

  it('renders children unchanged when nothing throws', () => {
    render(
      <ErrorBoundary>
        <Text>all good</Text>
      </ErrorBoundary>,
    );
    expect(screen.getByText('all good')).toBeTruthy();
    expect(screen.queryByText('Something went wrong')).toBeNull();
  });

  it('re-renders the subtree when "Try again" is pressed', () => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    let shouldThrow = true;
    const Maybe = () => {
      if (shouldThrow) throw new Error('boom');
      return <Text>recovered</Text>;
    };

    render(
      <ErrorBoundary>
        <Maybe />
      </ErrorBoundary>,
    );
    expect(screen.getByText('Something went wrong')).toBeTruthy();

    shouldThrow = false;
    fireEvent.press(screen.getByLabelText('Try again'));
    expect(screen.getByText('recovered')).toBeTruthy();

    jest.restoreAllMocks();
  });
});
