import React from 'react';
import Svg, { Path } from 'react-native-svg';

interface ThumbsDownIconProps {
  size?: number;
  color?: string;
}

export const ThumbsDownIcon: React.FC<ThumbsDownIconProps> = ({
  size = 24,
  color = '#0F0C0B'
}) => {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d="M17 2V13M22 11V4C22 3.46957 21.7893 2.96086 21.4142 2.58579C21.0391 2.21071 20.5304 2 20 2H7.72C7.23767 1.99451 6.76962 2.16359 6.40206 2.47599C6.03451 2.78839 5.79233 3.22305 5.72 3.7L4.34 12.7C4.29655 12.9866 4.31583 13.2793 4.39667 13.5577C4.47751 13.8362 4.61793 14.0937 4.80816 14.3125C4.99838 14.5313 5.23394 14.7061 5.49843 14.8248C5.76291 14.9435 6.05008 15.0033 6.34 15H10V19.5C10 20.163 10.2634 20.7989 10.7322 21.2678C11.2011 21.7366 11.837 22 12.5 22L17 13V2Z"
        stroke={color}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
};
