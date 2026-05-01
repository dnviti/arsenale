export function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

export function normalizeHex(hex: string) {
  const normalized = hex.replace('#', '').trim();
  if (normalized.length === 3) {
    return normalized
      .split('')
      .map((channel) => `${channel}${channel}`)
      .join('');
  }
  return normalized;
}

export function parseHex(hex: string) {
  const normalized = normalizeHex(hex);
  return {
    r: Number.parseInt(normalized.slice(0, 2), 16),
    g: Number.parseInt(normalized.slice(2, 4), 16),
    b: Number.parseInt(normalized.slice(4, 6), 16),
  };
}

export function rgbToHex(r: number, g: number, b: number) {
  return `#${[r, g, b]
    .map((channel) => clamp(Math.round(channel), 0, 255).toString(16).padStart(2, '0'))
    .join('')}`;
}

export function alpha(color: string, opacity: number) {
  if (color.startsWith('rgba(') || color.startsWith('hsla(')) {
    return color;
  }

  const { r, g, b } = parseHex(color);
  return `rgba(${r}, ${g}, ${b}, ${clamp(opacity, 0, 1)})`;
}

export function mixColors(foreground: string, background: string, ratio: number) {
  const fg = parseHex(foreground);
  const bg = parseHex(background);
  const weight = clamp(ratio, 0, 1);
  return rgbToHex(
    fg.r * weight + bg.r * (1 - weight),
    fg.g * weight + bg.g * (1 - weight),
    fg.b * weight + bg.b * (1 - weight),
  );
}

function linearize(channel: number) {
  const normalized = channel / 255;
  return normalized <= 0.03928
    ? normalized / 12.92
    : ((normalized + 0.055) / 1.055) ** 2.4;
}

export function relativeLuminance(color: string) {
  const { r, g, b } = parseHex(color);
  return (0.2126 * linearize(r)) + (0.7152 * linearize(g)) + (0.0722 * linearize(b));
}

export function getContrastText(background: string) {
  return relativeLuminance(background) > 0.5 ? '#09090b' : '#fafafa';
}

export function lighten(color: string, amount: number) {
  return mixColors('#ffffff', color, clamp(amount, 0, 1));
}

export function darken(color: string, amount: number) {
  return mixColors('#000000', color, clamp(amount, 0, 1));
}
