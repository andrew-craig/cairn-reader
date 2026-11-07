import React from 'react';
import Svg, { Ellipse, Path } from 'react-native-svg';
import { useColorScheme } from 'react-native';
import { Colors } from '../constants';

interface LogoMarkProps {
  size?: number;
  color?: string;
}

export const LogoMark: React.FC<LogoMarkProps> = ({ size = 100, color }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const fill = color ?? colors.brand;

  return (
    <Svg width={size} height={size} viewBox="0 0 1024 1024" fill="none">
      <Path d="M199.541 711.353C197.713 707.106 200.789 702 205.412 702L795 702V814L205.649 814C200.974 814 197.882 808.765 199.762 804.485C204.127 794.55 209.913 777.756 209.913 758.556C209.913 739.029 203.928 721.545 199.541 711.353Z" fill={fill} />
      <Ellipse cx="795" cy="758" rx="50" ry="56" fill={fill} />
      <Ellipse cx="733" cy="601" rx="50" ry="56" fill={fill} />
      <Path d="M266.33 553.683C264.917 549.556 267.951 545 272.313 545H733V657H272.488C268.082 657 265.029 652.351 266.485 648.192C269.928 638.359 274.682 621.22 274.682 601.556C274.682 581.594 269.783 563.768 266.33 553.683Z" fill={fill} />
      <Path d="M798.097 490.988C799.712 495.179 796.654 500 792.162 500H269V388H791.958C796.497 388 799.572 392.934 797.909 397.158C794.018 407.045 788.752 424.012 788.752 443.444C788.752 463.191 794.189 480.848 798.097 490.988Z" fill={fill} />
      <Ellipse cx="269" cy="444" rx="50" ry="56" fill={fill} />
      <Path d="M652.442 335.539C653.164 339.36 650.252 343 646.364 343H402V231H646.282C650.196 231 653.119 234.689 652.375 238.531C650.519 248.111 647.736 265.9 647.736 286.444C647.736 307.24 650.587 325.717 652.442 335.539Z" fill={fill} />
      <Ellipse cx="402" cy="287" rx="50" ry="56" fill={fill} />
    </Svg>
  );
};
