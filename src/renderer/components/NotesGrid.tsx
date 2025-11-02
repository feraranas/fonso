import React, { useMemo } from 'react';
import { Grid } from '@coleski/datagrid';
import { Note } from '../App.js';

interface NotesGridProps {
  notes: Note[];
  searchKeyword: string;
  onTagClick: (tag: string) => void;
  onDelete: (noteId: string) => Promise<void>;
  currentTagFilter: string;
}

export const NotesGrid: React.FC<NotesGridProps> = ({
  notes,
  searchKeyword,
  onTagClick,
  onDelete,
  currentTagFilter,
}) => {
  // Sort notes by creation date (newest first)
  const sortedNotes = useMemo(() => {
    return [...notes].sort(
      (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    );
  }, [notes]);

  // Transform Note objects to string[][] format required by Grid
  const rows = useMemo(() => {
    return sortedNotes.map(note => [
      note.title,
      note.url,
      (note.tags || []).join(', '),
      note.summary || '-',
      new Date(note.createdAt).toLocaleString(),
      note.id // Include ID for tracking
    ]);
  }, [sortedNotes]);

  // Define headers matching the row structure
  const headers = ['Title', 'URL', 'Tags', 'Summary', 'Created', 'ID'];

  // Handle empty states
  if (notes.length === 0) {
    if (searchKeyword) {
      return (
        <div id="notes-list" className="notes-list">
          <div className="empty-state">
            <p>
              No notes found matching "<strong>{searchKeyword}</strong>"
            </p>
            <p style={{ marginTop: '10px', fontSize: '0.9rem' }}>
              Try a different keyword
            </p>
          </div>
        </div>
      );
    } else {
      return (
        <div id="notes-list" className="notes-list">
          <div className="empty-state">
            <p>No notes yet. Add your first URL note!</p>
          </div>
        </div>
      );
    }
  }

  return (
    <div id="notes-list" className="notes-list" style={{ height: '600px' }}>
      <Grid
        rows={rows}
        headers={headers}
        draggableColumns
        fitToCanvas={false}
        minColumnWidth={140}
        ariaLabel="Notes grid"
      />
    </div>
  );
};

