// Renderer process script
// This runs in the renderer process and has access to the DOM

console.log('Renderer process loaded!');
console.log('Platform:', window.electronAPI.platform);
console.log('Electron version:', window.electronAPI.version);

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
});

/**
 * Initialize the application
 */
async function initializeApp() {
    setupForm();
    setupDateField();
    setupNotesListener();
    await loadAndDisplayNotes();
    console.log('App initialized successfully!');
}

/**
 * Set up the form event handlers
 */
function setupForm() {
    const form = document.getElementById('note-form');
    
    if (form) {
        form.addEventListener('submit', handleFormSubmit);
        console.log('Form event listener attached');
    }
}

/**
 * Set up the date field with current date/time
 */
function setupDateField() {
    const dateField = document.getElementById('date');
    
    if (dateField) {
        // Set current date/time as default
        const now = new Date();
        // Format for datetime-local input (YYYY-MM-DDTHH:MM)
        const formattedDate = now.toISOString().slice(0, 16);
        dateField.value = formattedDate;
        console.log('Date field initialized with current date');
    }
}

/**
 * Handle form submission
 * @param {Event} event - The form submit event
 */
async function handleFormSubmit(event) {
    event.preventDefault(); // Prevent page reload
    
    console.log('Form submitted!');
    
    // Get form data
    const formData = new FormData(event.target);
    const noteData = {
        url: formData.get('url'),
        title: formData.get('title'),
        summary: formData.get('summary'),
        date: formData.get('date')
    };
    
    console.log('Note data:', noteData);
    
    // Basic validation
    if (!noteData.url || !noteData.title) {
        showMessage('Please fill in the required fields (URL and Title)', 'error');
        return;
    }
    
    // Show loading state
    const submitButton = event.target.querySelector('button[type="submit"]');
    const originalText = submitButton.textContent;
    submitButton.textContent = '💾 Saving...';
    submitButton.disabled = true;
    
    try {
        // Save the note using IPC
        const result = await window.electronAPI.saveNote(noteData);
        
        if (result.success) {
            showMessage('Note saved successfully!', 'success');
            // Clear the form
            event.target.reset();
            setupDateField(); // Reset date to current time
        } else {
            showMessage(`Error saving note: ${result.error}`, 'error');
        }
    } catch (error) {
        console.error('Error saving note:', error);
        showMessage('Error saving note. Please try again.', 'error');
    } finally {
        // Restore button state
        submitButton.textContent = originalText;
        submitButton.disabled = false;
    }
}

/**
 * Set up listener for notes updates from main process
 */
function setupNotesListener() {
    window.electronAPI.onNotesUpdated(() => {
        console.log('Notes updated, reloading...');
        loadAndDisplayNotes();
    });
}

/**
 * Load and display all notes
 */
async function loadAndDisplayNotes() {
    try {
        const notes = await window.electronAPI.loadNotes();
        displayNotes(notes);
        updateNotesCount(notes.length);
    } catch (error) {
        console.error('Error loading notes:', error);
        showMessage('Error loading notes', 'error');
    }
}

/**
 * Display notes in the UI
 * @param {Array} notes - Array of note objects
 */
function displayNotes(notes) {
    const notesList = document.getElementById('notes-list');
    
    if (!notesList) {
        console.error('Notes list element not found');
        return;
    }
    
    if (notes.length === 0) {
        notesList.innerHTML = '<div class="empty-state"><p>No notes yet. Add your first URL note!</p></div>';
        return;
    }
    
    // Sort notes by creation date (newest first)
    const sortedNotes = notes.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
    
    // Generate HTML for each note
    const notesHTML = sortedNotes.map(note => createNoteCard(note)).join('');
    notesList.innerHTML = notesHTML;
    
    // Add event listeners to delete buttons
    addDeleteListeners();
}

/**
 * Create HTML for a single note card
 * @param {Object} note - Note object
 * @returns {string} HTML string
 */
function createNoteCard(note) {
    const createdDate = new Date(note.createdAt).toLocaleDateString();
    const createdTime = new Date(note.createdAt).toLocaleTimeString();
    
    return `
        <div class="note-card" data-note-id="${note.id}">
            <div class="note-header">
                <h3 class="note-title">
                    <a href="${note.url}" target="_blank" rel="noopener noreferrer">
                        ${escapeHtml(note.title)}
                    </a>
                </h3>
                <button class="btn-delete" data-note-id="${note.id}" title="Delete note">
                    🗑️
                </button>
            </div>
            <div class="note-url">
                <a href="${note.url}" target="_blank" rel="noopener noreferrer">
                    ${escapeHtml(note.url)}
                </a>
            </div>
            ${note.summary ? `<div class="note-summary">${escapeHtml(note.summary)}</div>` : ''}
            <div class="note-meta">
                <span class="note-date">Saved on ${createdDate} at ${createdTime}</span>
            </div>
        </div>
    `;
}

/**
 * Add event listeners to delete buttons
 */
function addDeleteListeners() {
    const deleteButtons = document.querySelectorAll('.btn-delete');
    deleteButtons.forEach(button => {
        button.addEventListener('click', async (event) => {
            const noteId = event.target.getAttribute('data-note-id');
            if (noteId && confirm('Are you sure you want to delete this note?')) {
                await deleteNote(noteId);
            }
        });
    });
}

/**
 * Delete a note
 * @param {string} noteId - ID of the note to delete
 */
async function deleteNote(noteId) {
    try {
        const result = await window.electronAPI.deleteNote(noteId);
        if (result.success) {
            showMessage('Note deleted successfully!', 'success');
        } else {
            showMessage(`Error deleting note: ${result.error}`, 'error');
        }
    } catch (error) {
        console.error('Error deleting note:', error);
        showMessage('Error deleting note. Please try again.', 'error');
    }
}

/**
 * Update the notes count display
 * @param {number} count - Number of notes
 */
function updateNotesCount(count) {
    const countElement = document.getElementById('notes-count');
    if (countElement) {
        countElement.textContent = count;
    }
}

/**
 * Show a message to the user
 * @param {string} message - Message to display
 * @param {string} type - Message type ('success', 'error', 'info')
 */
function showMessage(message, type = 'info') {
    // Create or update message element
    let messageElement = document.getElementById('message');
    if (!messageElement) {
        messageElement = document.createElement('div');
        messageElement.id = 'message';
        messageElement.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 12px 20px;
            border-radius: 8px;
            color: white;
            font-weight: 600;
            z-index: 1000;
            transition: all 0.3s ease;
        `;
        document.body.appendChild(messageElement);
    }
    
    // Set message content and style
    messageElement.textContent = message;
    messageElement.style.backgroundColor = type === 'success' ? '#28a745' : 
                                        type === 'error' ? '#dc3545' : '#17a2b8';
    
    // Show message
    messageElement.style.opacity = '1';
    messageElement.style.transform = 'translateY(0)';
    
    // Hide message after 3 seconds
    setTimeout(() => {
        messageElement.style.opacity = '0';
        messageElement.style.transform = 'translateY(-20px)';
    }, 3000);
}

/**
 * Escape HTML to prevent XSS
 * @param {string} text - Text to escape
 * @returns {string} Escaped text
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
