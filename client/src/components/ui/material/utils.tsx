import { forwardRef, type CSSProperties, type ReactNode, type SVGProps } from 'react';
import type { LucideIcon } from 'lucide-react';
import { resolveSx, useTheme, type SxProp } from './theme';

type MaterialIconProps = SVGProps<SVGSVGElement> & {
  color?: string;
  fontSize?: string;
  htmlColor?: string;
  sx?: SxProp;
  titleAccess?: string;
};

function resolveIconColor(theme: ReturnType<typeof useTheme>, color: string | undefined) {
  switch (color) {
    case 'action':
    case 'action.active':
    case 'text.secondary':
      return theme.palette.text.secondary;
    case 'disabled':
    case 'text.disabled':
      return theme.palette.text.disabled;
    case 'error':
    case 'error.main':
      return theme.palette.error.main;
    case 'info':
    case 'info.main':
      return theme.palette.info.main;
    case 'primary':
    case 'primary.main':
      return theme.palette.primary.main;
    case 'secondary':
    case 'secondary.main':
      return theme.palette.secondary.main;
    case 'success':
    case 'success.main':
      return theme.palette.success.main;
    case 'warning':
    case 'warning.main':
    case 'warning.light':
      return theme.palette.warning.main;
    case 'text.primary':
      return theme.palette.text.primary;
    case 'inherit':
    case 'default':
      return undefined;
    default:
      return color;
  }
}

function resolveIconSize(fontSize: unknown) {
  if (fontSize == null) {
    return '20px';
  }

  if (typeof fontSize === 'number') {
    return `${fontSize}px`;
  }

  switch (fontSize) {
    case 'inherit':
      return undefined;
    case 'small':
      return '16px';
    case 'large':
      return '24px';
    case 'medium':
      return '20px';
    default:
      return String(fontSize);
  }
}

function removeFontSize(style: CSSProperties | undefined) {
  if (!style) {
    return {};
  }

  const { fontSize: _fontSize, ...rest } = style;
  return rest;
}

function buildIconStyle(
  theme: ReturnType<typeof useTheme>,
  {
    color,
    fontSize,
    htmlColor,
    sx,
    style,
  }: Pick<MaterialIconProps, 'color' | 'fontSize' | 'htmlColor' | 'sx' | 'style'>,
) {
  const sxStyle = resolveSx(theme, sx) as CSSProperties | undefined;
  const resolvedColor = htmlColor ?? resolveIconColor(theme, color) ?? sxStyle?.color ?? style?.color;
  const explicitWidth = style?.width ?? sxStyle?.width;
  const explicitHeight = style?.height ?? sxStyle?.height;
  const resolvedSize = resolveIconSize(style?.fontSize ?? sxStyle?.fontSize ?? fontSize);

  return {
    style: {
      ...removeFontSize(sxStyle),
      ...removeFontSize(style),
      ...(resolvedColor ? { color: resolvedColor } : undefined),
      ...(resolvedSize && !explicitWidth ? { width: resolvedSize } : undefined),
      ...(resolvedSize && !explicitHeight ? { height: resolvedSize } : undefined),
    } satisfies CSSProperties,
  };
}

export function createSvgIcon(path: ReactNode, displayName: string) {
  const Component = forwardRef<SVGSVGElement, MaterialIconProps>(function SvgIcon(
    {
      color,
      fontSize,
      htmlColor,
      sx,
      style,
      titleAccess,
      viewBox = '0 0 24 24',
      ...props
    },
    ref,
  ) {
    const theme = useTheme();
    const { style: iconStyle } = buildIconStyle(theme, {
      color,
      fontSize,
      htmlColor,
      sx,
      style,
    });

    return (
      <svg
        ref={ref}
        viewBox={viewBox}
        fill="currentColor"
        role={titleAccess ? 'img' : undefined}
        aria-hidden={titleAccess ? undefined : true}
        style={iconStyle}
        {...props}
      >
        {titleAccess ? <title>{titleAccess}</title> : null}
        {path}
      </svg>
    );
  });

  Component.displayName = `${displayName}Icon`;
  return Component;
}

export function createLucideIcon(Icon: LucideIcon, displayName: string) {
  const Component = forwardRef<SVGSVGElement, MaterialIconProps>(function LucideAdapter(
    {
      color,
      fontSize,
      htmlColor,
      sx,
      style,
      titleAccess,
      ...props
    },
    ref,
  ) {
    const theme = useTheme();
    const { style: iconStyle } = buildIconStyle(theme, {
      color,
      fontSize,
      htmlColor,
      sx,
      style,
    });

    return (
      <Icon
        ref={ref}
        aria-label={titleAccess}
        aria-hidden={titleAccess ? undefined : true}
        role={titleAccess ? 'img' : undefined}
        style={iconStyle}
        {...props}
      />
    );
  });

  Component.displayName = `${displayName}Icon`;
  return Component;
}
