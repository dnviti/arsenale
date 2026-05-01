import { create } from 'zustand';

interface CommandPaletteState {
  open: boolean;
  toggle: () => void;
  setOpen: (open: boolean) => void;
}

export const useCommandPaletteStore = create<CommandPaletteState>((set) => ({
  open: false,
  toggle: () => set((s) => ({ open: !s.open })),
  setOpen: (open) => set({ open }),
}));
