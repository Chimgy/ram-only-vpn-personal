declare module 'react-simple-maps' {
  import { ComponentType, ReactNode, CSSProperties } from 'react';

  interface Geography {
    rsmKey: string;
    [key: string]: unknown;
  }

  interface GeographiesChildProps {
    geographies: Geography[];
  }

  interface GeographyStyle {
    default?: CSSProperties;
    hover?: CSSProperties;
    pressed?: CSSProperties;
  }

  export const ComposableMap: ComponentType<{
    projection?: string;
    projectionConfig?: { scale?: number; center?: [number, number] };
    style?: CSSProperties;
    [key: string]: unknown;
  }>;

  export const Geographies: ComponentType<{
    geography: string;
    children: (props: GeographiesChildProps) => ReactNode;
  }>;

  export const Geography: ComponentType<{
    geography: Geography;
    style?: GeographyStyle;
    fill?: string;
    stroke?: string;
    strokeWidth?: number;
    [key: string]: unknown;
  }>;

  export const Marker: ComponentType<{
    coordinates: [number, number];
    children?: ReactNode;
    [key: string]: unknown;
  }>;
}
