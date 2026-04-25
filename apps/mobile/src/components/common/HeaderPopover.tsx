import React from 'react';
import {
  Modal,
  View,
  TouchableOpacity,
  StyleSheet,
  Platform,
  KeyboardAvoidingView,
} from 'react-native';
import { BlurView } from 'expo-blur';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Colors, Spacing, Layout } from '../../constants/theme';

interface HeaderPopoverProps {
  visible: boolean;
  onClose: () => void;
  children: React.ReactNode;
}

export const HeaderPopover: React.FC<HeaderPopoverProps> = ({ visible, onClose, children }) => {
  const insets = useSafeAreaInsets();
  const topOffset = insets.top + Layout.headerHeight;

  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <View style={StyleSheet.absoluteFill}>
        <BlurView
          style={StyleSheet.absoluteFill}
          intensity={Platform.OS === 'ios' ? 8 : 16}
          tint="light"
        />
        <TouchableOpacity
          style={[styles.overlay, { paddingTop: topOffset }]}
          activeOpacity={1}
          onPress={onClose}
        >
          <KeyboardAvoidingView
            behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
            style={styles.keyboardView}
          >
            <TouchableOpacity
              activeOpacity={1}
              style={[styles.card, { backgroundColor: Colors.light.hover }]}
              onPress={(e) => e.stopPropagation()}
            >
              {children}
            </TouchableOpacity>
          </KeyboardAvoidingView>
        </TouchableOpacity>
      </View>
    </Modal>
  );
};

const styles = StyleSheet.create({
  overlay: {
    flex: 1,
    paddingHorizontal: Spacing.md,
    justifyContent: 'flex-start',
  },
  keyboardView: {
    width: '100%',
  },
  card: {
    borderRadius: 21,
    padding: Spacing.md,
    gap: Spacing.md,
    shadowColor: 'rgba(15, 23, 42, 0.2)',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 1,
    shadowRadius: 4,
    elevation: 4,
  },
});
