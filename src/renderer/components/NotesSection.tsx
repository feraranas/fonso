import React from 'react';
import { SearchBar } from './SearchBar.js';
import { NotesGrid } from './NotesGrid.js';
import { Note } from '../App.js';

interface NotesSectionProps {
  notes: Note[];
  filteredNotes: Note[];
  searchKeyword: string;
  tagFilter: string;
  loading: boolean;
  onSearch: (keyword: string) => void;
  onTagFilter: (tag: string) => void;
  onClearTagFilter: () => void;
  onClearSearch: () => void;
  onExport: () => Promise<void>;
  onImport: () => Promise<void>;
  onDelete: (noteId: string) => Promise<void>;
}

export const NotesSection: React.FC<NotesSectionProps> = ({
  notes,
  filteredNotes,
  searchKeyword,
  tagFilter,
  loading,
  onSearch,
  onTagFilter,
  onClearTagFilter,
  onClearSearch,
  onExport,
  onImport,
  onDelete,
}) => {
  const notesCount = filteredNotes.length;
  const totalCount = notes.length;

  return (
    <div className="notes-section">
      <div className="notes-header">
        <h2>Your Saved Notes</h2>
        <div className="header-actions">
          <div className="notes-count">
            <span id="notes-count">
              {searchKeyword && notesCount !== totalCount
                ? `${notesCount} of ${totalCount} notes`
                : `${notesCount} note${notesCount !== 1 ? 's' : ''}`}
            </span>
          </div>
          <div className="export-import-buttons">
            <button
              id="export-btn"
              className="btn btn-export"
              title="Export notes to JSON file"
              onClick={onExport}
            >
              📤 Export
            </button>
            <button
              id="import-btn"
              className="btn btn-import"
              title="Import notes from JSON file"
              onClick={onImport}
            >
              📥 Import
            </button>
          </div>
        </div>
        
        {tagFilter && (
          <div className="tag-filter-indicator">
            <span>Filtered by:</span>
            <span className="tag-name">{tagFilter}</span>
            <button
              className="btn-clear-tag-filter"
              title="Clear tag filter"
              onClick={onClearTagFilter}
            >
              ✕
            </button>
          </div>
        )}
      </div>

      <SearchBar
        value={searchKeyword}
        onChange={onSearch}
        onClear={onClearSearch}
      />

      <NotesGrid
        notes={filteredNotes}
        searchKeyword={searchKeyword}
        onTagClick={onTagFilter}
        onDelete={onDelete}
        currentTagFilter={tagFilter}
      />
    </div>
  );
};

