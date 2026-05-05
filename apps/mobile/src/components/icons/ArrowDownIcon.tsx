import React from 'react';
import Svg, { Path } from 'react-native-svg';

interface ArrowDownIconProps {
  size?: number;
  color?: string;
}

export const ArrowDownIcon: React.FC<ArrowDownIconProps> = ({
  size = 24,
  color = '#0F0C0B'
}) => {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d="M12 5v14M5 12l7 7 7-7"
        stroke={color}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
};
