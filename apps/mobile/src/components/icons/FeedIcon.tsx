import React from 'react';
import Svg, { Path } from 'react-native-svg';

interface FeedIconProps {
  size?: number;
  color?: string;
  filled?: boolean;
}

export const FeedIcon: React.FC<FeedIconProps> = ({
  size = 24,
  color = '#0F0C0B',
  filled = false,
}) => {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill={filled ? color : 'none'} stroke={color}>
      <Path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
        d="M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 0 1 0 3.75H5.625a1.875 1.875 0 0 1 0-3.75Z"
      />
    </Svg>
  );
};
