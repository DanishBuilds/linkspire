// Tiny inline sparkline. Decorative (aria-hidden); the number it sits next to
// carries the meaning. Server-renderable.
export function Sparkline({ data, width = 96, height = 28 }: { data: number[]; width?: number; height?: number }) {
  if (!data || data.length < 2) return null;
  const max = Math.max(1, ...data);
  const min = Math.min(...data);
  const span = Math.max(1, max - min);
  const step = width / (data.length - 1);
  const y = (v: number) => height - 2 - ((v - min) / span) * (height - 4);
  const line = data.map((v, i) => `${i === 0 ? 'M' : 'L'}${(i * step).toFixed(1)},${y(v).toFixed(1)}`).join(' ');
  const area = `${line} L${width},${height} L0,${height} Z`;
  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" aria-hidden="true">
      <path d={area} fill="var(--accent-soft)" opacity="0.6" />
      <path d={line} fill="none" stroke="var(--c1)" strokeWidth="1.8" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}
