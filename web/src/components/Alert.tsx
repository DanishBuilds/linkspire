// src/components/Alert.tsx — inline message block. role=alert for errors so
// assistive tech announces them; status for neutral info.
import type { ReactNode } from 'react';
import { Icon } from './icons';

type Variant = 'info' | 'success' | 'warning' | 'danger';
const icons: Record<Variant, ReactNode> = {
  info: <Icon.Info className="ico" />, success: <Icon.Check className="ico" />,
  warning: <Icon.Alert className="ico" />, danger: <Icon.Alert className="ico" />,
};

export function Alert({
  variant = 'info', title, children,
}: { variant?: Variant; title?: string; children?: ReactNode }) {
  const assertive = variant === 'danger' || variant === 'warning';
  return (
    <div className={`alert ${variant}`} role={assertive ? 'alert' : 'status'}>
      {icons[variant]}
      <div>
        {title ? <strong>{title}</strong> : null}
        {children ? <div>{children}</div> : null}
      </div>
    </div>
  );
}
