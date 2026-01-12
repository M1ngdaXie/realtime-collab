import type { CSSProperties } from 'react';

interface FloatingGemProps {
  position?: { top?: string; right?: string; bottom?: string; left?: string };
  delay?: number;
  size?: number;
}

const FloatingGem = ({ position = {}, delay = 0, size = 32 }: FloatingGemProps) => {
  const style: CSSProperties = {
    position: 'absolute',
    ...position,
    animation: `pixel-float 4s ease-in-out infinite`,
    animationDelay: `${delay}s`,
    pointerEvents: 'none',
    zIndex: 10,
  };

  return (
    <svg
      width={size}
      height={size}
      style={style}
      viewBox="0 0 24 24"
      aria-hidden="true"
    >
      <use href="/icons/pixel-sprites.svg#icon-gem" />
    </svg>
  );
};

export default FloatingGem;
