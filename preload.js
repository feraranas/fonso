const { contextBridge, ipcRenderer } = require('electron');

// Expose protected methods that allow the renderer process to use
// the ipcRenderer without exposing the entire object
contextBridge.exposeInMainWorld('electronAPI', {
    // System information
    platform: process.platform,
    version: process.versions.electron,
    
    // Note management methods
    saveNote: (noteData) => ipcRenderer.invoke('save-note', noteData),
    loadNotes: () => ipcRenderer.invoke('load-notes'),
    deleteNote: (noteId) => ipcRenderer.invoke('delete-note', noteId),
    
    // Export/Import methods
    exportNotes: () => ipcRenderer.invoke('export-notes'),
    importNotes: (options) => ipcRenderer.invoke('import-notes', options),
    
    // Event listeners for real-time updates
    onNotesUpdated: (callback) => {
        ipcRenderer.on('notes-updated', callback);
    },
    
    // Remove event listeners
    removeAllListeners: (channel) => {
        ipcRenderer.removeAllListeners(channel);
    }
});
