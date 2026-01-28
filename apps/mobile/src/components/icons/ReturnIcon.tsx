import React from 'react';
import Svg, { Path } from 'react-native-svg';

interface ReturnIconProps {
  size?: number;
  color?: string;
}

export const ReturnIcon: React.FC<ReturnIconProps> = ({
  size = 24,
  color = '#0F0C0B'
}) => {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d="M19 12H5M12 19L5 12L12 5"
        stroke={color}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
};
