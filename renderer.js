// Renderer process script
// This runs in the renderer process and has access to the DOM

console.log('Renderer process loaded!');
console.log('Platform:', window.electronAPI.platform);
console.log('Electron version:', window.electronAPI.version);

// State management for notes
let allNotes = []; // Store all loaded notes
let currentSearchKeyword = ''; // Track current search keyword
let currentTagFilter = ''; // Track currently selected tag filter

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
    setupSearch();
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
 * Parse comma-separated tags string into array
 * @param {string} tagString - Comma-separated tags string
 * @returns {Array<string>} Array of normalized tags
 */
function parseTags(tagString) {
    if (!tagString || !tagString.trim()) {
        return [];
    }
    
    return tagString
        .split(',')
        .map(tag => normalizeTag(tag))
        .filter(tag => tag.length > 0); // Remove empty tags
}

/**
 * Normalize a tag (trim whitespace, convert to lowercase)
 * @param {string} tag - Tag to normalize
 * @returns {string} Normalized tag
 */
function normalizeTag(tag) {
    return tag.trim().toLowerCase();
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
    const tagsString = formData.get('tags') || '';
    const noteData = {
        url: formData.get('url'),
        title: formData.get('title'),
        summary: formData.get('summary'),
        date: formData.get('date'),
        tags: parseTags(tagsString) // Parse tags from comma-separated string
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
        // Reload notes and maintain current search state
        loadAndDisplayNotes();
    });
}

/**
 * Load and display all notes
 */
async function loadAndDisplayNotes() {
    try {
        const notes = await window.electronAPI.loadNotes();
        // Store all notes in state
        allNotes = notes;
        // Apply current search filter if any
        applySearchFilter();
    } catch (error) {
        console.error('Error loading notes:', error);
        showMessage('Error loading notes', 'error');
    }
}

/**
 * Set up search functionality
 */
function setupSearch() {
    const searchInput = document.getElementById('search-input');
    const clearButton = document.getElementById('clear-search-btn');
    
    if (searchInput) {
        // Real-time search as user types
        searchInput.addEventListener('input', handleSearchInput);
        
        // Handle Enter key to focus on results
        searchInput.addEventListener('keydown', (event) => {
            if (event.key === 'Escape') {
                clearSearch();
            }
        });
    }
    
    if (clearButton) {
        clearButton.addEventListener('click', clearSearch);
    }
}

/**
 * Handle search input changes
 * @param {Event} event - The input event
 */
function handleSearchInput(event) {
    const keyword = event.target.value.trim();
    currentSearchKeyword = keyword;
    
    // Show/hide clear button based on input
    const clearButton = document.getElementById('clear-search-btn');
    if (clearButton) {
        clearButton.style.display = keyword ? 'flex' : 'none';
    }
    
    // Apply search filter
    applySearchFilter();
}

/**
 * Filter notes by keyword and tag
 * @param {string} keyword - Search keyword
 * @param {string} tagFilter - Tag to filter by
 * @returns {Array} Filtered notes
 */
function filterNotes(keyword, tagFilter) {
    let filtered = allNotes;
    
    // Apply tag filter first
    if (tagFilter) {
        const normalizedTag = normalizeTag(tagFilter);
        filtered = filtered.filter(note => {
            // Ensure backward compatibility - notes without tags array default to empty array
            const tags = note.tags || [];
            return tags.some(tag => normalizeTag(tag) === normalizedTag);
        });
    }
    
    // Apply keyword search
    if (keyword) {
        const lowerKeyword = keyword.toLowerCase();
        filtered = filtered.filter(note => {
            // Search in title
            const titleMatch = note.title && note.title.toLowerCase().includes(lowerKeyword);
            
            // Search in URL
            const urlMatch = note.url && note.url.toLowerCase().includes(lowerKeyword);
            
            // Search in summary
            const summaryMatch = note.summary && note.summary.toLowerCase().includes(lowerKeyword);
            
            // Search in tags
            const tags = note.tags || [];
            const tagsMatch = tags.some(tag => tag.toLowerCase().includes(lowerKeyword));
            
            return titleMatch || urlMatch || summaryMatch || tagsMatch;
        });
    }
    
    return filtered;
}

/**
 * Apply search filter and tag filter, then update display
 */
function applySearchFilter() {
    const filteredNotes = filterNotes(currentSearchKeyword, currentTagFilter);
    displayNotes(filteredNotes);
    updateNotesCount(filteredNotes.length, allNotes.length);
    updateTagFilterIndicator();
}

/**
 * Filter notes by tag
 * @param {string} tag - Tag to filter by
 */
function filterByTag(tag) {
    currentTagFilter = tag;
    applySearchFilter();
}

/**
 * Clear tag filter
 */
function clearTagFilter() {
    currentTagFilter = '';
    applySearchFilter();
}

/**
 * Clear search and show all notes
 */
function clearSearch() {
    const searchInput = document.getElementById('search-input');
    const clearButton = document.getElementById('clear-search-btn');
    
    if (searchInput) {
        searchInput.value = '';
        currentSearchKeyword = '';
    }
    
    if (clearButton) {
        clearButton.style.display = 'none';
    }
    
    // Display all notes (tag filter remains active)
    applySearchFilter();
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
    
    // Handle empty states
    if (notes.length === 0) {
        if (currentSearchKeyword) {
            // Show "no results" message when searching
            notesList.innerHTML = `
                <div class="empty-state">
                    <p>No notes found matching "<strong>${escapeHtml(currentSearchKeyword)}</strong>"</p>
                    <p style="margin-top: 10px; font-size: 0.9rem;">Try a different keyword or <button id="clear-search-empty-btn" style="background: none; border: none; color: #667eea; cursor: pointer; text-decoration: underline;">clear the search</button></p>
                </div>
            `;
            // Add event listener to the clear button in empty state
            const clearEmptyBtn = document.getElementById('clear-search-empty-btn');
            if (clearEmptyBtn) {
                clearEmptyBtn.addEventListener('click', clearSearch);
            }
        } else {
            // Show "no notes" message when there are no notes at all
            notesList.innerHTML = '<div class="empty-state"><p>No notes yet. Add your first URL note!</p></div>';
        }
        return;
    }
    
    // Sort notes by creation date (newest first)
    const sortedNotes = [...notes].sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
    
    // Generate HTML for each note
    const notesHTML = sortedNotes.map(note => createNoteCard(note)).join('');
    notesList.innerHTML = notesHTML;
    
    // Add event listeners to delete buttons and tag badges
    addDeleteListeners();
    addTagListeners();
}

/**
 * Create HTML for a single note card
 * @param {Object} note - Note object
 * @returns {string} HTML string
 */
function createNoteCard(note) {
    const createdDate = new Date(note.createdAt).toLocaleDateString();
    const createdTime = new Date(note.createdAt).toLocaleTimeString();
    const tags = note.tags || [];
    
    // Generate tags HTML
    const tagsHTML = tags.length > 0 
        ? `<div class="note-tags">${tags.map(tag => createTagBadge(tag, true)).join('')}</div>`
        : '';
    
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
            ${tagsHTML}
            ${note.summary ? `<div class="note-summary">${escapeHtml(note.summary)}</div>` : ''}
            <div class="note-meta">
                <span class="note-date">Saved on ${createdDate} at ${createdTime}</span>
            </div>
        </div>
    `;
}

/**
 * Create HTML for a tag badge
 * @param {string} tag - Tag name
 * @param {boolean} isClickable - Whether the tag should be clickable
 * @returns {string} HTML string for tag badge
 */
function createTagBadge(tag, isClickable = true) {
    const isActive = currentTagFilter && normalizeTag(currentTagFilter) === normalizeTag(tag);
    const activeClass = isActive ? 'active' : '';
    
    if (isClickable) {
        return `<button class="tag-badge ${activeClass}" data-tag="${escapeHtml(tag)}" title="Filter by ${escapeHtml(tag)}">${escapeHtml(tag)}</button>`;
    } else {
        return `<span class="tag-badge">${escapeHtml(tag)}</span>`;
    }
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
 * Add event listeners to tag badges
 */
function addTagListeners() {
    const tagBadges = document.querySelectorAll('.tag-badge[data-tag]');
    tagBadges.forEach(badge => {
        badge.addEventListener('click', (event) => {
            const tag = event.target.getAttribute('data-tag');
            if (tag) {
                // Toggle tag filter: if same tag clicked, clear filter; otherwise set new filter
                if (currentTagFilter && normalizeTag(currentTagFilter) === normalizeTag(tag)) {
                    clearTagFilter();
                } else {
                    filterByTag(tag);
                }
            }
        });
    });
}

/**
 * Update tag filter indicator in the UI
 */
function updateTagFilterIndicator() {
    const notesHeader = document.querySelector('.notes-header');
    if (!notesHeader) return;
    
    // Remove existing indicator
    const existingIndicator = document.getElementById('tag-filter-indicator');
    if (existingIndicator) {
        existingIndicator.remove();
    }
    
    // Add indicator if tag filter is active
    if (currentTagFilter) {
        const indicator = document.createElement('div');
        indicator.id = 'tag-filter-indicator';
        indicator.className = 'tag-filter-indicator';
        indicator.innerHTML = `
            <span>Filtered by:</span>
            <span class="tag-name">${escapeHtml(currentTagFilter)}</span>
            <button class="btn-clear-tag-filter" title="Clear tag filter">✕</button>
        `;
        
        // Add click handler to clear button
        const clearBtn = indicator.querySelector('.btn-clear-tag-filter');
        if (clearBtn) {
            clearBtn.addEventListener('click', clearTagFilter);
        }
        
        // Append to notes header
        if (notesHeader) {
            notesHeader.appendChild(indicator);
        }
    }
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
 * @param {number} filteredCount - Number of filtered notes
 * @param {number} totalCount - Total number of notes
 */
function updateNotesCount(filteredCount, totalCount) {
    const countElement = document.getElementById('notes-count');
    if (countElement) {
        if (currentSearchKeyword && filteredCount !== totalCount) {
            // Show "X of Y notes" when filtered
            countElement.textContent = `${filteredCount} of ${totalCount} notes`;
        } else {
            // Show just the count when not searching or all notes match
            countElement.textContent = `${filteredCount} note${filteredCount !== 1 ? 's' : ''}`;
        }
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
