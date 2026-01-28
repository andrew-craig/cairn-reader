import React from 'react';
import Svg, { Path } from 'react-native-svg';

interface ThumbsUpIconProps {
  size?: number;
  color?: string;
}

export const ThumbsUpIcon: React.FC<ThumbsUpIconProps> = ({
  size = 24,
  color = '#0F0C0B'
}) => {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d="M7 22V11M2 13V20C2 20.5304 2.21071 21.0391 2.58579 21.4142C2.96086 21.7893 3.46957 22 4 22H16.28C16.7623 22.0055 17.2304 21.8364 17.5979 21.524C17.9654 21.2116 18.2077 20.7769 18.28 20.3L19.66 11.3C19.7035 11.0134 19.6842 10.7207 19.6033 10.4423C19.5225 10.1638 19.3821 9.90629 19.1919 9.68751C19.0016 9.46873 18.7661 9.29393 18.5016 9.17522C18.2371 9.0565 17.9499 8.99672 17.66 9H14V4.5C14 3.83696 13.7366 3.20107 13.2678 2.73223C12.7989 2.26339 12.163 2 11.5 2L7 11V22Z"
        stroke={color}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
};
