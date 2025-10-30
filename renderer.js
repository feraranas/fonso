// Renderer process script
// This runs in the renderer process and has access to the DOM

console.log('Renderer process loaded!');
console.log('Checking window.electronAPI:', window.electronAPI);
if (window.electronAPI) {
    console.log('Platform:', window.electronAPI.platform);
    console.log('Electron version:', window.electronAPI.version);
    console.log('loadNotes method available:', typeof window.electronAPI.loadNotes);
} else {
    console.error('ERROR: window.electronAPI is not defined!');
}

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
    initializeTheme();
    setupForm();
    setupDateField();
    setupSearch();
    setupExportImport();
    setupThemeToggle();
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
        // Set current date/time as default (using local time, not UTC)
        const now = new Date();
        // Format for datetime-local input (YYYY-MM-DDTHH:MM)
        // Use local time components instead of UTC to avoid timezone offset issues
        const year = now.getFullYear();
        const month = String(now.getMonth() + 1).padStart(2, '0');
        const day = String(now.getDate()).padStart(2, '0');
        const hours = String(now.getHours()).padStart(2, '0');
        const minutes = String(now.getMinutes()).padStart(2, '0');
        const formattedDate = `${year}-${month}-${day}T${hours}:${minutes}`;
        dateField.value = formattedDate;
        console.log('Date field initialized with current local date/time');
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
 * Initialize theme from localStorage or default to light
 */
function initializeTheme() {
    const savedTheme = localStorage.getItem('theme') || 'light';
    setTheme(savedTheme);
}

/**
 * Set theme and update UI
 * @param {string} theme - 'light' or 'dark'
 */
function setTheme(theme) {
    console.log('Setting theme to:', theme);
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
    updateThemeIcon(theme);
    console.log('Theme set successfully. data-theme attribute:', document.documentElement.getAttribute('data-theme'));
}

/**
 * Toggle between light and dark theme
 */
function toggleTheme() {
    console.log('Theme toggle clicked');
    const currentTheme = document.documentElement.getAttribute('data-theme') || 'light';
    console.log('Current theme:', currentTheme);
    const newTheme = currentTheme === 'light' ? 'dark' : 'light';
    console.log('Switching to theme:', newTheme);
    setTheme(newTheme);
}

/**
 * Update theme icon based on current theme
 * @param {string} theme - Current theme
 */
function updateThemeIcon(theme) {
    const themeIcon = document.querySelector('.theme-icon');
    if (themeIcon) {
        themeIcon.textContent = theme === 'dark' ? '☀️' : '🌙';
    }
}

/**
 * Set up theme toggle button
 */
function setupThemeToggle() {
    const themeToggle = document.getElementById('theme-toggle');
    if (themeToggle) {
        themeToggle.addEventListener('click', toggleTheme);
        console.log('Theme toggle button event listener attached');
    } else {
        console.error('Theme toggle button not found!');
    }
}

/**
 * Set up export/import functionality
 */
function setupExportImport() {
    const exportBtn = document.getElementById('export-btn');
    const importBtn = document.getElementById('import-btn');
    
    if (exportBtn) {
        exportBtn.addEventListener('click', handleExport);
    }
    
    if (importBtn) {
        importBtn.addEventListener('click', handleImport);
    }
}

/**
 * Handle export button click
 */
async function handleExport() {
    const exportBtn = document.getElementById('export-btn');
    if (!exportBtn) return;
    
    // Disable button during export
    const originalText = exportBtn.textContent;
    exportBtn.textContent = '📤 Exporting...';
    exportBtn.disabled = true;
    
    try {
        const result = await window.electronAPI.exportNotes();
        
        if (result.canceled) {
            // User canceled, no message needed
            return;
        }
        
        if (result.success) {
            const fileName = result.filePath.split(/[/\\]/).pop();
            showMessage(`Notes exported successfully to ${fileName}`, 'success');
        } else {
            showMessage(`Export failed: ${result.error}`, 'error');
        }
    } catch (error) {
        console.error('Error exporting notes:', error);
        showMessage('Error exporting notes. Please try again.', 'error');
    } finally {
        // Restore button state
        exportBtn.textContent = originalText;
        exportBtn.disabled = false;
    }
}

/**
 * Handle import button click
 */
async function handleImport() {
    const importBtn = document.getElementById('import-btn');
    if (!importBtn) return;
    
    // Check if there are existing notes
    const hasExistingNotes = allNotes.length > 0;
    
    let replaceMode = false;
    if (hasExistingNotes) {
        // Ask user if they want to merge or replace
        const userChoice = confirm(
            'Do you want to REPLACE all existing notes with imported notes?\n\n' +
            'Click OK to REPLACE all notes\n' +
            'Click Cancel to MERGE (add new notes, keep existing ones)'
        );
        replaceMode = userChoice;
    }
    
    // Disable button during import
    const originalText = importBtn.textContent;
    importBtn.textContent = '📥 Importing...';
    importBtn.disabled = true;
    
    try {
        const result = await window.electronAPI.importNotes({ replace: replaceMode });
        
        if (result.canceled) {
            // User canceled, no message needed
            return;
        }
        
        if (result.success) {
            let message = `Successfully imported ${result.imported} note${result.imported !== 1 ? 's' : ''}`;
            if (result.skipped > 0) {
                message += ` (${result.skipped} invalid note${result.skipped !== 1 ? 's' : ''} skipped)`;
            }
            if (result.total !== undefined) {
                message += `. Total notes: ${result.total}`;
            }
            showMessage(message, 'success');
        } else {
            showMessage(`Import failed: ${result.error}`, 'error');
        }
    } catch (error) {
        console.error('Error importing notes:', error);
        showMessage('Error importing notes. Please check the file format.', 'error');
    } finally {
        // Restore button state
        importBtn.textContent = originalText;
        importBtn.disabled = false;
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
    console.log('loadAndDisplayNotes: Starting to load notes...');
    try {
        const notes = await window.electronAPI.loadNotes();
        console.log('loadAndDisplayNotes: Notes loaded from API:', notes);
        console.log('loadAndDisplayNotes: Number of notes:', notes ? notes.length : 0);
        // Store all notes in state
        allNotes = notes;
        console.log('loadAndDisplayNotes: allNotes updated, length:', allNotes.length);
        // Apply current search filter if any
        applySearchFilter();
        console.log('loadAndDisplayNotes: applySearchFilter called');
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
    console.log('applySearchFilter: Called. currentSearchKeyword:', currentSearchKeyword, 'currentTagFilter:', currentTagFilter);
    console.log('applySearchFilter: allNotes.length:', allNotes.length);
    const filteredNotes = filterNotes(currentSearchKeyword, currentTagFilter);
    console.log('applySearchFilter: filteredNotes.length:', filteredNotes.length);
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
    console.log('displayNotes: Called with', notes ? notes.length : 0, 'notes');
    const notesList = document.getElementById('notes-list');
    
    if (!notesList) {
        console.error('displayNotes: Notes list element not found!');
        return;
    }
    console.log('displayNotes: notesList element found');
    
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
    console.log('displayNotes: Sorting notes...');
    const sortedNotes = [...notes].sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
    console.log('displayNotes: Sorted', sortedNotes.length, 'notes');
    
    // Generate HTML for each note
    console.log('displayNotes: Generating HTML...');
    const notesHTML = sortedNotes.map(note => createNoteCard(note)).join('');
    console.log('displayNotes: Generated HTML length:', notesHTML.length);
    notesList.innerHTML = notesHTML;
    console.log('displayNotes: notesList.innerHTML set');
    
    // Add event listeners to delete buttons and tag badges
    addDeleteListeners();
    addTagListeners();
    console.log('displayNotes: Event listeners added');
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
    
    // Sanitize URL for safe use in href attributes
    const safeUrl = sanitizeUrl(note.url);
    
    // Generate tags HTML
    const tagsHTML = tags.length > 0 
        ? `<div class="note-tags">${tags.map(tag => createTagBadge(tag, true)).join('')}</div>`
        : '';
    
    return `
        <div class="note-card" data-note-id="${escapeHtml(note.id)}">
            <div class="note-header">
                <h3 class="note-title">
                    <a href="${safeUrl}" target="_blank" rel="noopener noreferrer">
                        ${escapeHtml(note.title)}
                    </a>
                </h3>
                <button class="btn-delete" data-note-id="${escapeHtml(note.id)}" title="Delete note">
                    🗑️
                </button>
            </div>
            <div class="note-url">
                <a href="${safeUrl}" target="_blank" rel="noopener noreferrer">
                    ${escapeHtml(note.url)}
                </a>
            </div>
            ${tagsHTML}
            ${note.summary ? `<div class="note-summary">${escapeHtml(note.summary)}</div>` : ''}
            <div class="note-meta">
                <span class="note-date">Saved on ${escapeHtml(createdDate)} at ${escapeHtml(createdTime)}</span>
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
            box-shadow: 0 4px 12px var(--shadow-dark);
        `;
        document.body.appendChild(messageElement);
    }
    
    // Set message content and style
    messageElement.textContent = message;
    const rootStyles = getComputedStyle(document.documentElement);
    messageElement.style.backgroundColor = type === 'success' ? rootStyles.getPropertyValue('--success-color').trim() : 
                                        type === 'error' ? rootStyles.getPropertyValue('--error-color').trim() : 
                                        rootStyles.getPropertyValue('--info-color').trim();
    
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

/**
 * Sanitize URL for use in href attribute
 * Validates URL scheme and returns safe URL or about:blank for dangerous URLs
 * @param {string} url - URL to sanitize
 * @returns {string} Sanitized URL safe for href attribute (HTML-escaped)
 */
function sanitizeUrl(url) {
    if (!url || typeof url !== 'string') {
        return 'about:blank';
    }
    
    // Trim whitespace
    url = url.trim();
    
    // Check for dangerous protocols (case-insensitive)
    const dangerousProtocols = ['javascript:', 'data:', 'vbscript:', 'file:', 'about:'];
    const lowerUrl = url.toLowerCase();
    
    for (const protocol of dangerousProtocols) {
        if (lowerUrl.startsWith(protocol)) {
            console.warn('Blocked dangerous URL protocol:', url);
            return 'about:blank';
        }
    }
    
    // Allow http, https, mailto, tel, and relative URLs
    const allowedPatterns = [
        /^https?:\/\//i,           // http:// or https://
        /^mailto:/i,               // mailto:
        /^tel:/i,                  // tel:
        /^\//,                     // Absolute path
        /^\.\//,                   // Relative path ./ 
        /^\.\.\//                  // Relative path ../
    ];
    
    const hasAllowedPattern = allowedPatterns.some(pattern => pattern.test(url));
    
    // If no protocol and doesn't look like a relative path, try to make it absolute
    if (!hasAllowedPattern) {
        // If it looks like a domain, add https://
        const domainPattern = /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+/;
        if (domainPattern.test(url.split('/')[0].split('?')[0].split('#')[0])) {
            url = 'https://' + url;
        } else {
            // Not a valid absolute or relative URL, block it
            console.warn('Blocked invalid URL format:', url);
            return 'about:blank';
        }
    }
    
    // Validate the URL and ensure it uses only allowed protocols
    try {
        // Try to parse as absolute URL
        const urlObj = new URL(url, window.location.href);
        const allowedProtocols = ['http:', 'https:', 'mailto:', 'tel:'];
        
        if (allowedProtocols.includes(urlObj.protocol)) {
            // Escape HTML special characters for safe use in href attribute
            return escapeHtml(urlObj.href);
        } else {
            console.warn('Blocked non-allowed protocol:', urlObj.protocol);
            return 'about:blank';
        }
    } catch (e) {
        // If URL parsing fails, check if it's a relative URL
        if (url.startsWith('/') || url.startsWith('./') || url.startsWith('../')) {
            // Relative URLs are safe, but still escape HTML special characters
            return escapeHtml(url);
        } else {
            console.warn('Failed to parse URL:', url, e);
            return 'about:blank';
        }
    }
}
