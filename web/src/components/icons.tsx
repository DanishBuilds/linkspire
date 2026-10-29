// src/components/icons.tsx — inline, currentColor SVG icons. Decorative by
// default (aria-hidden); pass a title for a labelled icon.
import type { SVGProps } from 'react';

type IconProps = SVGProps<SVGSVGElement> & { title?: string };

function base({ title, children, ...props }: IconProps & { children: React.ReactNode }) {
  return (
    <svg
      width={18} height={18} viewBox="0 0 24 24" fill="none"
      stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"
      aria-hidden={title ? undefined : true} role={title ? 'img' : undefined}
      focusable="false" {...props}
    >
      {title ? <title>{title}</title> : null}
      {children}
    </svg>
  );
}

export const Icon = {
  Dashboard: (p: IconProps) => base({ ...p, children: <><rect x="3" y="3" width="7" height="9" rx="1" /><rect x="14" y="3" width="7" height="5" rx="1" /><rect x="14" y="12" width="7" height="9" rx="1" /><rect x="3" y="16" width="7" height="5" rx="1" /></> }),
  Link: (p: IconProps) => base({ ...p, children: <><path d="M9 15 15 9" /><path d="M11 6l1-1a4 4 0 0 1 6 6l-1 1" /><path d="M13 18l-1 1a4 4 0 0 1-6-6l1-1" /></> }),
  Globe: (p: IconProps) => base({ ...p, children: <><circle cx="12" cy="12" r="9" /><path d="M3 12h18M12 3c2.5 2.5 2.5 15 0 18M12 3c-2.5 2.5-2.5 15 0 18" /></> }),
  Key: (p: IconProps) => base({ ...p, children: <><circle cx="8" cy="15" r="4" /><path d="M11 12 20 3M17 6l2 2M14 9l2 2" /></> }),
  Webhook: (p: IconProps) => base({ ...p, children: <><path d="M9 7a3 3 0 1 1 4 2.8l2.5 4.2" /><path d="M7.5 12 5 16a3 3 0 1 0 3 4h5" /><path d="M16 11a3 3 0 1 1 0 6h-5" /></> }),
  Settings: (p: IconProps) => base({ ...p, children: <><circle cx="12" cy="12" r="3" /><path d="M12 2v3M12 19v3M2 12h3M19 12h3M5 5l2 2M17 17l2 2M5 19l2-2M17 7l2-2" /></> }),
  Shield: (p: IconProps) => base({ ...p, children: <><path d="M12 3l7 3v5c0 4.5-3 8-7 10-4-2-7-5.5-7-10V6z" /><path d="m9 12 2 2 4-4" /></> }),
  Chart: (p: IconProps) => base({ ...p, children: <><path d="M4 20V10M10 20V4M16 20v-7M22 20H2" /></> }),
  Copy: (p: IconProps) => base({ ...p, children: <><rect x="9" y="9" width="12" height="12" rx="2" /><path d="M5 15V5a2 2 0 0 1 2-2h8" /></> }),
  Check: (p: IconProps) => base({ ...p, children: <path d="m5 13 4 4L19 7" /> }),
  Plus: (p: IconProps) => base({ ...p, children: <path d="M12 5v14M5 12h14" /> }),
  Trash: (p: IconProps) => base({ ...p, children: <><path d="M4 7h16M10 11v6M14 11v6" /><path d="M6 7l1 13h10l1-13M9 7V4h6v3" /></> }),
  External: (p: IconProps) => base({ ...p, children: <><path d="M14 4h6v6M20 4l-9 9" /><path d="M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5" /></> }),
  Logout: (p: IconProps) => base({ ...p, children: <><path d="M15 4h4a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1h-4" /><path d="M10 12H3M6 8l-3 4 3 4" /></> }),
  Alert: (p: IconProps) => base({ ...p, children: <><path d="M12 3 2 20h20z" /><path d="M12 10v4M12 17h.01" /></> }),
  Info: (p: IconProps) => base({ ...p, children: <><circle cx="12" cy="12" r="9" /><path d="M12 11v5M12 8h.01" /></> }),
  Lock: (p: IconProps) => base({ ...p, children: <><rect x="4" y="10" width="16" height="11" rx="2" /><path d="M8 10V7a4 4 0 0 1 8 0v3" /></> }),
  Sun: (p: IconProps) => base({ ...p, children: <><circle cx="12" cy="12" r="4" /><path d="M12 2v2M12 20v2M2 12h2M20 12h2M5 5l1.5 1.5M17.5 17.5 19 19M5 19l1.5-1.5M17.5 6.5 19 5" /></> }),
  Moon: (p: IconProps) => base({ ...p, children: <path d="M21 12.8A8 8 0 1 1 11.2 3a6 6 0 0 0 9.8 9.8z" /> }),
};
