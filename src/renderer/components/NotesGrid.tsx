import React, { useMemo } from 'react';
import { Grid } from '@coleski/datagrid';
import { Note } from '../App.js';
import { escapeHtml, sanitizeUrl } from '../utils/sanitize.js';

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

  const handleDelete = async (noteId: string) => {
    if (window.confirm('Are you sure you want to delete this note?')) {
      await onDelete(noteId);
    }
  };

  const normalizeTag = (tag: string) => tag.trim().toLowerCase();

  // Prepare columns for the datagrid
  const columns = [
    {
      key: 'title',
      name: 'Title',
      width: 250,
      resizable: true,
      sortable: true,
      formatter: ({ row }: { row: Note }) => {
        const safeUrl = sanitizeUrl(row.url);
        return (
          <a
            href={safeUrl}
            target="_blank"
            rel="noopener noreferrer"
            style={{ color: 'inherit', textDecoration: 'none', fontWeight: '600' }}
          >
            {row.title}
          </a>
        );
      },
    },
    {
      key: 'url',
      name: 'URL',
      width: 300,
      resizable: true,
      sortable: true,
      formatter: ({ row }: { row: Note }) => {
        const safeUrl = sanitizeUrl(row.url);
        return (
          <a
            href={safeUrl}
            target="_blank"
            rel="noopener noreferrer"
            style={{
              color: 'var(--link-color)',
              textDecoration: 'none',
              fontSize: '0.9rem',
            }}
          >
            {row.url}
          </a>
        );
      },
    },
    {
      key: 'tags',
      name: 'Tags',
      width: 200,
      resizable: true,
      formatter: ({ row }: { row: Note }) => {
        const tags = row.tags || [];
        return (
          <div style={{ display: 'flex', gap: '4px', flexWrap: 'wrap' }}>
            {tags.map((tag, index) => {
              const isActive =
                currentTagFilter && normalizeTag(currentTagFilter) === normalizeTag(tag);
              return (
                <button
                  key={index}
                  className={`tag-badge ${isActive ? 'active' : ''}`}
                  onClick={(e) => {
                    e.stopPropagation();
                    if (isActive) {
                      onTagClick(''); // Clear filter
                    } else {
                      onTagClick(tag);
                    }
                  }}
                  title={`Filter by ${tag}`}
                  style={{
                    padding: '2px 8px',
                    fontSize: '0.75rem',
                    borderRadius: '12px',
                    border: 'none',
                    cursor: 'pointer',
                    backgroundColor: isActive ? 'var(--primary-color)' : 'var(--tag-bg)',
                    color: isActive ? 'white' : 'var(--tag-color)',
                  }}
                >
                  {tag}
                </button>
              );
            })}
          </div>
        );
      },
    },
    {
      key: 'summary',
      name: 'Summary',
      width: 300,
      resizable: true,
      formatter: ({ row }: { row: Note }) => {
        return (
          <div style={{ fontSize: '0.9rem', color: 'var(--text-secondary)' }}>
            {row.summary || '-'}
          </div>
        );
      },
    },
    {
      key: 'createdAt',
      name: 'Created',
      width: 180,
      resizable: true,
      sortable: true,
      formatter: ({ row }: { row: Note }) => {
        const date = new Date(row.createdAt);
        return (
          <div style={{ fontSize: '0.85rem' }}>
            {date.toLocaleDateString()} {date.toLocaleTimeString()}
          </div>
        );
      },
    },
    {
      key: 'actions',
      name: 'Actions',
      width: 80,
      formatter: ({ row }: { row: Note }) => {
        return (
          <button
            className="btn-delete"
            onClick={(e) => {
              e.stopPropagation();
              handleDelete(row.id);
            }}
            title="Delete note"
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              fontSize: '1.2rem',
              padding: '4px',
            }}
          >
            🗑️
          </button>
        );
      },
    },
  ];

  return (
    <div id="notes-list" className="notes-list" style={{ height: '600px' }}>
      <Grid
        columns={columns}
        rows={sortedNotes}
        rowKeyGetter={(row: Note) => row.id}
        style={{ height: '100%' }}
      />
    </div>
  );
};

