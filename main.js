const { app, BrowserWindow, ipcMain } = require('electron');
const path = require('path');
const fs = require('fs').promises;

// Keep a global reference of the window object
let mainWindow;

// Data storage configuration
const DATA_DIR = path.join(__dirname, 'data');
const NOTES_FILE = path.join(DATA_DIR, 'notes.json');

/**
 * Initialize data storage
 * Creates the data directory and initializes the notes file if it doesn't exist
 */
async function initializeDataStorage() {
    try {
        // Create data directory if it doesn't exist
        await fs.mkdir(DATA_DIR, { recursive: true });
        
        // Initialize notes file if it doesn't exist
        try {
            await fs.access(NOTES_FILE);
        } catch (error) {
            // File doesn't exist, create it with empty array
            await fs.writeFile(NOTES_FILE, JSON.stringify([], null, 2));
            console.log('Created new notes file');
        }
        
        console.log('Data storage initialized');
    } catch (error) {
        console.error('Error initializing data storage:', error);
    }
}

/**
 * Load all notes from the JSON file
 * @returns {Promise<Array>} Array of note objects
 */
async function loadNotes() {
    try {
        const data = await fs.readFile(NOTES_FILE, 'utf8');
        return JSON.parse(data);
    } catch (error) {
        console.error('Error loading notes:', error);
        return [];
    }
}

/**
 * Save all notes to the JSON file
 * @param {Array} notes - Array of note objects
 * @returns {Promise<boolean>} Success status
 */
async function saveNotes(notes) {
    try {
        await fs.writeFile(NOTES_FILE, JSON.stringify(notes, null, 2));
        return true;
    } catch (error) {
        console.error('Error saving notes:', error);
        return false;
    }
}

/**
 * Generate a unique ID for a note
 * @returns {string} Unique identifier
 */
function generateNoteId() {
    return Date.now().toString(36) + Math.random().toString(36).substr(2);
}

/**
 * Notify renderer process that notes have been updated
 */
function notifyNotesUpdated() {
    if (mainWindow) {
        mainWindow.webContents.send('notes-updated');
    }
}

/**
 * Set up IPC handlers for communication with renderer process
 */
function setupIpcHandlers() {
    // Handle loading all notes
    ipcMain.handle('load-notes', async () => {
        console.log('Loading notes...');
        const notes = await loadNotes();
        // Ensure backward compatibility: add empty tags array to notes without tags
        return notes.map(note => ({
            ...note,
            tags: note.tags || []
        }));
    });

    // Handle saving a new note
    ipcMain.handle('save-note', async (event, noteData) => {
        console.log('Saving note:', noteData);
        
        try {
            // Load existing notes
            const notes = await loadNotes();
            
            // Add new note with ID and timestamp
            // Ensure tags is always an array for backward compatibility
            const newNote = {
                id: generateNoteId(),
                ...noteData,
                tags: noteData.tags || [],
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString()
            };
            
            // Add to notes array
            notes.push(newNote);
            
            // Save back to file
            const success = await saveNotes(notes);
            
            if (success) {
                // Notify renderer that notes were updated
                notifyNotesUpdated();
                return { success: true, note: newNote };
            } else {
                return { success: false, error: 'Failed to save note' };
            }
        } catch (error) {
            console.error('Error saving note:', error);
            return { success: false, error: error.message };
        }
    });

    // Handle deleting a note
    ipcMain.handle('delete-note', async (event, noteId) => {
        console.log('Deleting note:', noteId);
        
        try {
            // Load existing notes
            const notes = await loadNotes();
            
            // Filter out the note to delete
            const updatedNotes = notes.filter(note => note.id !== noteId);
            
            // Save back to file
            const success = await saveNotes(updatedNotes);
            
            if (success) {
                // Notify renderer that notes were updated
                notifyNotesUpdated();
                return { success: true };
            } else {
                return { success: false, error: 'Failed to delete note' };
            }
        } catch (error) {
            console.error('Error deleting note:', error);
            return { success: false, error: error.message };
        }
    });
}

/**
 * Create the main application window
 * This function sets up the BrowserWindow with appropriate settings
 */
const createWindow = () => {
  // Create the browser window
  mainWindow = new BrowserWindow({
    width: 1000,
    height: 700,
    minWidth: 800,
    minHeight: 600,
    webPreferences: {
      // Enable Node.js integration in the renderer process
      nodeIntegration: false, // Security best practice
      contextIsolation: true, // Security best practice
      enableRemoteModule: false, // Security best practice
      preload: path.join(__dirname, 'preload.js') // We'll create this file next
    },
    // Add a title bar with custom styling
    titleBarStyle: 'hiddenInset', // macOS style
    show: false // Don't show until ready
  });

  // Load the HTML file
  mainWindow.loadFile('index.html');

  // Show window when ready to prevent visual flash
  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
  });

  // Open DevTools in development
  if (process.env.NODE_ENV === 'development') {
    mainWindow.webContents.openDevTools();
  }

  // Emitted when the window is closed
  mainWindow.on('closed', () => {
    mainWindow = null;
  });
};

// This method will be called when Electron has finished initialization
// and is ready to create browser windows
app.whenReady().then(async () => {
  // Initialize data storage first
  await initializeDataStorage();
  
  // Set up IPC handlers
  setupIpcHandlers();
  
  // Create the main window
  createWindow();

  // On macOS, it's common to re-create a window when the dock icon is clicked
  // and there are no other windows open
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

// Quit when all windows are closed (except on macOS)
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

// Security: Prevent new window creation
app.on('web-contents-created', (event, contents) => {
  contents.on('new-window', (event, navigationUrl) => {
    event.preventDefault();
  });
});
