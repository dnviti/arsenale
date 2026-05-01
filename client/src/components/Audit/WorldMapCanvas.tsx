import type { CSSProperties, ReactNode } from 'react';
import { MapContainer, type MapContainerProps } from 'react-leaflet';
import { WorldBasemap } from './WorldBasemap';

const worldMapCanvasStyle: CSSProperties = {
  width: '100%',
  height: '100%',
  background:
    'radial-gradient(circle at top, rgba(15, 23, 42, 0.16), rgba(15, 23, 42, 0.04) 38%, rgba(248, 250, 252, 0.96) 100%)',
};

interface WorldMapCanvasProps extends MapContainerProps {
  children?: ReactNode;
  minHeight?: CSSProperties['minHeight'];
}

export function WorldMapCanvas({
  children,
  minHeight,
  scrollWheelZoom = true,
  style,
  ...props
}: WorldMapCanvasProps) {
  return (
    <MapContainer
      {...props}
      scrollWheelZoom={scrollWheelZoom}
      style={{
        ...worldMapCanvasStyle,
        minHeight,
        ...style,
      }}
    >
      <WorldBasemap />
      {children}
    </MapContainer>
  );
}
