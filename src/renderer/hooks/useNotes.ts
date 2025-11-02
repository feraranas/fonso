import { useState, useEffect, useCallback } from 'react';
import { Note } from '../App.js';
import { showMessage } from '../utils/message.js';

export const useNotes = () => {
  const [notes, setNotes] = useState<Note[]>([]);
  const [filteredNotes, setFilteredNotes] = useState<Note[]>([]);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [tagFilter, setTagFilter] = useState('');
  const [loading, setLoading] = useState(true);

  const normalizeTag = (tag: string) => tag.trim().toLowerCase();

  const filterNotes = useCallback(
    (allNotes: Note[], keyword: string, tag: string) => {
      let filtered = allNotes;

      // Apply tag filter
      if (tag) {
        const normalizedTag = normalizeTag(tag);
        filtered = filtered.filter((note) => {
          const tags = note.tags || [];
          return tags.some((t) => normalizeTag(t) === normalizedTag);
        });
      }

      // Apply keyword search
      if (keyword) {
        const lowerKeyword = keyword.toLowerCase();
        filtered = filtered.filter((note) => {
          const titleMatch = note.title?.toLowerCase().includes(lowerKeyword);
          const urlMatch = note.url?.toLowerCase().includes(lowerKeyword);
          const summaryMatch = note.summary?.toLowerCase().includes(lowerKeyword);
          const tags = note.tags || [];
          const tagsMatch = tags.some((tag) => tag.toLowerCase().includes(lowerKeyword));
          return titleMatch || urlMatch || summaryMatch || tagsMatch;
        });
      }

      return filtered;
    },
    []
  );

  const loadNotes = useCallback(async () => {
    try {
      const loadedNotes = await window.electronAPI.loadNotes();
      setNotes(loadedNotes);
      const filtered = filterNotes(loadedNotes, searchKeyword, tagFilter);
      setFilteredNotes(filtered);
    } catch (error) {
      console.error('Error loading notes:', error);
      showMessage('Error loading notes', 'error');
    } finally {
      setLoading(false);
    }
  }, [searchKeyword, tagFilter, filterNotes]);

  useEffect(() => {
    loadNotes();

    // Listen for notes updates
    window.electronAPI.onNotesUpdated(() => {
      console.log('Notes updated, reloading...');
      loadNotes();
    });
  }, [loadNotes]);

  const handleSearch = useCallback(
    (keyword: string) => {
      setSearchKeyword(keyword);
      const filtered = filterNotes(notes, keyword, tagFilter);
      setFilteredNotes(filtered);
    },
    [notes, tagFilter, filterNotes]
  );

  const handleTagFilter = useCallback(
    (tag: string) => {
      // Toggle: if same tag, clear it; otherwise set new tag
      if (tagFilter && normalizeTag(tagFilter) === normalizeTag(tag)) {
        setTagFilter('');
        const filtered = filterNotes(notes, searchKeyword, '');
        setFilteredNotes(filtered);
      } else {
        setTagFilter(tag);
        const filtered = filterNotes(notes, searchKeyword, tag);
        setFilteredNotes(filtered);
      }
    },
    [notes, searchKeyword, tagFilter, filterNotes]
  );

  const clearTagFilter = useCallback(() => {
    setTagFilter('');
    const filtered = filterNotes(notes, searchKeyword, '');
    setFilteredNotes(filtered);
  }, [notes, searchKeyword, filterNotes]);

  const clearSearch = useCallback(() => {
    setSearchKeyword('');
    const filtered = filterNotes(notes, '', tagFilter);
    setFilteredNotes(filtered);
  }, [notes, tagFilter, filterNotes]);

  const handleExport = async () => {
    try {
      const result = await window.electronAPI.exportNotes();

      if (result.canceled) {
        return;
      }

      if (result.success && result.filePath) {
        const fileName = result.filePath.split(/[/\\]/).pop();
        showMessage(`Notes exported successfully to ${fileName}`, 'success');
      } else {
        showMessage(`Export failed: ${result.error}`, 'error');
      }
    } catch (error) {
      console.error('Error exporting notes:', error);
      showMessage('Error exporting notes. Please try again.', 'error');
    }
  };

  const handleImport = async () => {
    const hasExistingNotes = notes.length > 0;

    let replaceMode = false;
    if (hasExistingNotes) {
      const userChoice = window.confirm(
        'Do you want to REPLACE all existing notes with imported notes?\n\n' +
          'Click OK to REPLACE all notes\n' +
          'Click Cancel to MERGE (add new notes, keep existing ones)'
      );
      replaceMode = userChoice;
    }

    try {
      const result = await window.electronAPI.importNotes({ replace: replaceMode });

      if (result.canceled) {
        return;
      }

      if (result.success) {
        let message = `Successfully imported ${result.imported} note${
          result.imported !== 1 ? 's' : ''
        }`;
        if (result.skipped && result.skipped > 0) {
          message += ` (${result.skipped} invalid note${
            result.skipped !== 1 ? 's' : ''
          } skipped)`;
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
    }
  };

  const handleDelete = async (noteId: string) => {
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
  };

  return {
    notes,
    filteredNotes,
    searchKeyword,
    tagFilter,
    loading,
    handleSearch,
    handleTagFilter,
    clearTagFilter,
    clearSearch,
    handleExport,
    handleImport,
    handleDelete,
  };
};

