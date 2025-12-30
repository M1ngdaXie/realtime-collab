interface PixelIconProps {
  name: string;
  size?: number;
  color?: string;
  animated?: boolean;
  className?: string;
  style?: React.CSSProperties;
}

const PixelIcon = ({
  name,
  size = 24,
  color,
  animated = false,
  className = '',
  style = {}
}: PixelIconProps) => {
  return (
    <svg
      width={size}
      height={size}
      className={`pixel-icon ${animated ? 'pixel-icon-animated' : ''} ${className}`}
      style={{ color, ...style }}
      aria-hidden="true"
    >
      <use href={`/icons/pixel-sprites.svg#icon-${name}`} />
    </svg>
  );
};

export default PixelIcon;
