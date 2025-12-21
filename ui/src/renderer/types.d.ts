export interface ElectronAPI {
  platform: string;
  version: string;
  loadNotes: () => Promise<Note[]>;
  saveNote: (note: NoteData) => Promise<{ success: boolean; error?: string }>;
  deleteNote: (noteId: string) => Promise<{ success: boolean; error?: string }>;
  exportNotes: () => Promise<{ success: boolean; error?: string; filePath?: string; canceled?: boolean }>;
  importNotes: (options: { replace: boolean }) => Promise<{ 
    success: boolean; 
    error?: string; 
    imported?: number; 
    skipped?: number; 
    total?: number;
    canceled?: boolean;
  }>;
  onNotesUpdated: (callback: () => void) => void;
}

export interface Note {
  id: string;
  url: string;
  title: string;
  summary?: string;
  date?: string;
  tags: string[];
  createdAt: string;
}

export interface NoteData {
  url: string;
  title: string;
  summary?: string;
  date?: string;
  tags: string[];
}

declare global {
  interface Window {
    electronAPI: ElectronAPI;
  }
}

export {};

