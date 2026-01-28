import React from 'react';
import Svg, { Path } from 'react-native-svg';

interface ArchiveIconProps {
  size?: number;
  color?: string;
}

export const ArchiveIcon: React.FC<ArchiveIconProps> = ({
  size = 24,
  color = '#0F0C0B'
}) => {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d="M21 8V21H3V8M1 3H23V8H1V3ZM10 12H14"
        stroke={color}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
};
