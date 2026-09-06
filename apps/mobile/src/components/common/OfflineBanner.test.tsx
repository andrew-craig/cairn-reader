import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { OfflineBanner } from './OfflineBanner';
import { useNetworkStatus } from '../../hooks/useNetworkStatus';

jest.mock('../../hooks/useNetworkStatus');

describe('OfflineBanner', () => {
  afterEach(() => jest.clearAllMocks());

  it('renders nothing when online', () => {
    (useNetworkStatus as jest.Mock).mockReturnValue({ isOffline: false });
    render(<OfflineBanner />);
    expect(screen.queryByText(/offline/i)).toBeNull();
  });

  it('renders a message when offline', () => {
    (useNetworkStatus as jest.Mock).mockReturnValue({ isOffline: true });
    render(<OfflineBanner />);
    expect(screen.getByText(/offline/i)).toBeTruthy();
  });
});
