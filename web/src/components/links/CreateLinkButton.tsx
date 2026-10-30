'use client';

import { useState } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { Plus, X } from 'lucide-react';
import { CreateLinkForm } from '@/components/links/CreateLinkForm';
import type { DomainRecord } from '@/lib/types';

// "Create link" trigger that opens a right-side slide-over containing the
// create form. The form and its server action are unchanged.
export function CreateLinkButton({ domains, label = 'Create link' }: { domains: DomainRecord[]; label?: string }) {
  const [open, setOpen] = useState(false);
  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <button type="button" className="btn"><Plus className="h-4 w-4" />{label}</button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="slideover-overlay" />
        <Dialog.Content className="slideover" aria-describedby={undefined}>
          <div className="slideover-head">
            <Dialog.Title className="slideover-title">Create a link</Dialog.Title>
            <Dialog.Close asChild>
              <button type="button" className="icon-btn" aria-label="Close"><X className="h-4 w-4" /></button>
            </Dialog.Close>
          </div>
          <div className="slideover-body">
            <CreateLinkForm domains={domains} canWrite />
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
