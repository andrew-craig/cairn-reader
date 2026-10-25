interface CloseIconProps {
  size?: number;
}

// Ionicons `close-outline`. Colour follows `currentColor`.
export default function CloseIcon({ size = 24 }: CloseIconProps) {
  return (
    <svg
      aria-hidden="true"
      width={size}
      height={size}
      viewBox="0 0 512 512"
      fill="none"
      stroke="currentColor"
      strokeWidth="32"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M368 368L144 144M368 144L144 368" />
    </svg>
  );
}
