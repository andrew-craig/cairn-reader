import React from 'react';
import Svg, { Path, Circle } from 'react-native-svg';
import { useColorScheme } from 'react-native';
import { Colors } from '../constants';

interface LogoMarkProps {
  size?: number;
  color?: string;
}

export const LogoMark: React.FC<LogoMarkProps> = ({ size = 100, color }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const fill = color ?? colors.primary;

  return (
    <Svg width={size} height={size} viewBox="0 0 600 600" fill="none">
      <Path d="M78 471C78 467.686 80.6863 465 84 465H498V533H84C80.6863 533 78 530.314 78 527V471Z" fill={fill} />
      <Circle cx="498" cy="499" r="34" fill={fill} />
      <Path d="M117 383C117 379.686 119.686 377 123 377H459V445H123C119.686 445 117 442.314 117 439V383Z" fill={fill} />
      <Circle cx="459" cy="411" r="34" fill={fill} />
      <Path d="M156 207C156 203.686 158.686 201 162 201H420V269H162C158.686 269 156 266.314 156 263V207Z" fill={fill} />
      <Circle cx="420" cy="235" r="34" fill={fill} />
      <Path d="M486 351C486 354.314 483.314 357 480 357L152 357V289L480 289C483.314 289 486 291.686 486 295V351Z" fill={fill} />
      <Circle cx="152" cy="323" r="34" fill={fill} />
      <Path d="M406 175C406 178.314 403.314 181 400 181L231 181L231 113L400 113C403.314 113 406 115.686 406 119V175Z" fill={fill} />
      <Circle cx="231" cy="147" r="34" fill={fill} />
    </Svg>
  );
};
