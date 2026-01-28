import React from 'react';
import Svg, { Path } from 'react-native-svg';

interface BookmarkIconProps {
  size?: number;
  color?: string;
}

export const BookmarkIcon: React.FC<BookmarkIconProps> = ({
  size = 24,
  color = '#0F0C0B'
}) => {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d="M5 3C4.46957 3 3.96086 3.21071 3.58579 3.58579C3.21071 3.96086 3 4.46957 3 5V21L12 17L21 21V5C21 4.46957 20.7893 3.96086 20.4142 3.58579C20.0391 3.21071 19.5304 3 19 3H5Z"
        stroke={color}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
};
