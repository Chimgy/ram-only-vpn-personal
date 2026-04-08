import { ComposableMap, Geographies, Geography, Marker } from 'react-simple-maps';

const GEO_URL = 'https://cdn.jsdelivr.net/npm/world-atlas@2/countries-110m.json';

// Change to your Pi's actual coordinates
const SERVER_COORDS: [number, number] = [-0.1276, 51.5074];

interface Props { connected: boolean; }

const dimmedStyle = {
  default: { fill: '#1e1b4b', stroke: '#0f0d2e', strokeWidth: 0.4, outline: 'none' },
  hover:   { fill: '#1e1b4b', stroke: '#0f0d2e', strokeWidth: 0.4, outline: 'none' },
  pressed: { fill: '#1e1b4b', stroke: '#0f0d2e', strokeWidth: 0.4, outline: 'none' },
};

const brightStyle = {
  default: { fill: '#4f46e5', stroke: '#818cf8', strokeWidth: 0.5, outline: 'none' },
  hover:   { fill: '#4f46e5', stroke: '#818cf8', strokeWidth: 0.5, outline: 'none' },
  pressed: { fill: '#4f46e5', stroke: '#818cf8', strokeWidth: 0.5, outline: 'none' },
};

export default function MapBackground({ connected }: Props) {
  return (
    <div className="absolute inset-0">
      {/* Dimmed base map */}
      <div className="absolute inset-0 opacity-40">
        <ComposableMap
          projection="geoMercator"
          projectionConfig={{ scale: 140, center: [0, 20] }}
          style={{ width: '100%', height: '100%' }}
        >
          <Geographies geography={GEO_URL}>
            {({ geographies }) =>
              geographies.map(geo => (
                <Geography key={geo.rsmKey} geography={geo} style={dimmedStyle} />
              ))
            }
          </Geographies>
        </ComposableMap>
      </div>

      {/* Bright revealed map — clip-path driven by CSS vars, no React re-renders */}
      <div className="map-lens absolute inset-0 opacity-90">
        <ComposableMap
          projection="geoMercator"
          projectionConfig={{ scale: 140, center: [0, 20] }}
          style={{ width: '100%', height: '100%' }}
        >
          <Geographies geography={GEO_URL}>
            {({ geographies }) =>
              geographies.map(geo => (
                <Geography key={geo.rsmKey} geography={geo} style={brightStyle} />
              ))
            }
          </Geographies>
          <Marker coordinates={SERVER_COORDS}>
            <circle r={4} fill={connected ? '#4ade80' : '#f59e0b'} />
            <circle r={8} fill={connected ? '#4ade8030' : '#f59e0b25'} />
          </Marker>
        </ComposableMap>
      </div>
    </div>
  );
}
