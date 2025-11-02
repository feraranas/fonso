import React, { useRef, useEffect, useMemo } from 'react';
import { Grid, html } from 'gridjs';
import 'gridjs/dist/theme/mermaid.css';
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
  // Refs
  const wrapperRef = useRef<HTMLDivElement>(null);
  const gridInstanceRef = useRef<Grid | null>(null);

  // Helper function
  const normalizeTag = (tag: string) => tag.trim().toLowerCase();

  // Sort and prepare data
  const sortedNotes = useMemo(() => {
    return [...notes].sort(
      (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    );
  }, [notes]);

  // Prepare grid data
  const gridData = useMemo(() => {
    return sortedNotes.map((note) => {
      const safeUrl = sanitizeUrl(note.url);
      const date = new Date(note.createdAt);
      
      // Title column with link
      const titleHtml = html(
        `<a href="${escapeHtml(safeUrl)}" target="_blank" rel="noopener noreferrer" style="color: inherit; text-decoration: none; font-weight: 600;">${escapeHtml(note.title)}</a>`
      );
      
      // URL column with link
      const urlHtml = html(
        `<a href="${escapeHtml(safeUrl)}" target="_blank" rel="noopener noreferrer" style="color: var(--link-color); text-decoration: none; font-size: 0.9rem;">${escapeHtml(note.url)}</a>`
      );
      
      // Tags column with clickable badges
      const tagsHtml = html(
        (note.tags || [])
          .map((tag) => {
            const isActive =
              currentTagFilter && normalizeTag(currentTagFilter) === normalizeTag(tag);
            return `<button 
              class="tag-badge ${isActive ? 'active' : ''}" 
              data-tag="${escapeHtml(tag)}"
              style="padding: 2px 8px; font-size: 0.75rem; border-radius: 12px; border: none; cursor: pointer; background-color: ${isActive ? 'var(--primary-color)' : 'var(--tag-bg)'}; color: ${isActive ? 'white' : 'var(--tag-color)'}; margin: 2px;">
              ${escapeHtml(tag)}
            </button>`;
          })
          .join('') || '-'
      );
      
      // Summary column
      const summaryHtml = html(
        `<div style="font-size: 0.9rem; color: var(--text-secondary);">${escapeHtml(note.summary || '-')}</div>`
      );
      
      // Created date column
      const dateHtml = html(
        `<div style="font-size: 0.85rem;">${date.toLocaleDateString()} ${date.toLocaleTimeString()}</div>`
      );
      
      // Actions column with delete button
      const actionsHtml = html(
        `<button 
          class="btn-delete" 
          data-note-id="${note.id}"
          style="background: none; border: none; cursor: pointer; font-size: 1.2rem; padding: 4px;"
          title="Delete note">
          🗑️
        </button>`
      );
      
      return [titleHtml, urlHtml, tagsHtml, summaryHtml, dateHtml, actionsHtml];
    });
  }, [sortedNotes, currentTagFilter]);

  // Render grid with useEffect
  useEffect(() => {
    if (!wrapperRef.current) return;

    // Destroy existing grid if it exists
    if (gridInstanceRef.current) {
      gridInstanceRef.current.destroy();
      gridInstanceRef.current = null;
    }

    // Don't render grid if no notes
    if (notes.length === 0) return;

    // Create new grid instance
    const grid = new Grid({
      columns: [
      {
        name: "Title",
        sort: true,
        width: '10%'
      },
      {
        name: "URL",
        width: '20%'
      },
      {
        name: "Tags",
        width: '10%'
      },
      {
        name: "Summary",
        width: '40%'
      },
      {
        name: "Created",
        width: '10%'
      },
      {
        name: "Actions",
        width: '10%'
      }],
      data: gridData,
      pagination: { limit: 20 },
      sort: true,
      resizable: true,
      search: false,
    });

    gridInstanceRef.current = grid;
    grid.render(wrapperRef.current);

    // Event delegation for clicks
    const handleClick = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      
      // Handle tag clicks
      if (target.classList.contains('tag-badge')) {
        e.stopPropagation();
        const tag = target.getAttribute('data-tag');
        if (tag) {
          const isActive =
            currentTagFilter && normalizeTag(currentTagFilter) === normalizeTag(tag);
          if (isActive) {
            onTagClick(''); // Clear filter
          } else {
            onTagClick(tag);
          }
        }
      }
      
      // Handle delete button clicks
      if (target.classList.contains('btn-delete') || target.closest('.btn-delete')) {
        e.stopPropagation();
        const deleteBtn = target.classList.contains('btn-delete') 
          ? target 
          : target.closest('.btn-delete') as HTMLElement;
        const noteId = deleteBtn?.getAttribute('data-note-id');
        if (noteId) {
          if (window.confirm('Are you sure you want to delete this note?')) {
            onDelete(noteId);
          }
        }
      }
    };

    wrapperRef.current.addEventListener('click', handleClick);

    // Cleanup
    return () => {
      if (wrapperRef.current) {
        wrapperRef.current.removeEventListener('click', handleClick);
      }
      if (gridInstanceRef.current) {
        gridInstanceRef.current.destroy();
        gridInstanceRef.current = null;
      }
    };
  }, [gridData, notes.length, currentTagFilter, onTagClick, onDelete]);

  // Empty state - after all hooks
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
      <div ref={wrapperRef} style={{ height: '100%' }}></div>
    </div>
  );
};

