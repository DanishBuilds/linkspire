'use client';

const items = [
  { id: 'link-details', label: 'Details' },
  { id: 'qr-code', label: 'QR code' },
  { id: 'link-analytics', label: 'Analytics' },
] as const;

export function LinkSubnav() {
  return (
    <nav className="link-subnav" aria-label="Link sections">
      {items.map((item) => (
        <a key={item.id} href={`#${item.id}`}>{item.label}</a>
      ))}
    </nav>
  );
}
