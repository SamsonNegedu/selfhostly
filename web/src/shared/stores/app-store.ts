import { create } from 'zustand';
import { App } from '../types/api';

interface AppStore {
  apps: App[];
  selectedApp: App | null;
  setApps: (apps: App[]) => void;
  addApp: (app: App) => void;
  updateApp: (id: string, app: Partial<App>) => void;
  removeApp: (id: string) => void;
  selectApp: (app: App | null) => void;
}

export const useAppStore = create<AppStore>((set) => ({
  apps: [],
  selectedApp: null,
  setApps: (apps) => set({ apps }),
  addApp: (app) => set((state) => ({ apps: [app, ...state.apps] })),
  updateApp: (id, updates) =>
    set((state) => ({
      apps: state.apps.map((app) =>
        app.id === id ? { ...app, ...updates } : app
      ),
      selectedApp:
        state.selectedApp?.id === id
          ? { ...state.selectedApp, ...updates }
          : state.selectedApp,
    })),
  removeApp: (id) =>
    set((state) => ({
      apps: state.apps.filter((app) => app.id !== id),
      selectedApp: state.selectedApp?.id === id ? null : state.selectedApp,
    })),
  selectApp: (app) => set({ selectedApp: app }),
}));
