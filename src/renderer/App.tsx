import React, { useState, useEffect } from 'react';
import { Header } from './components/Header.js';
import { NoteForm } from './components/NoteForm.js';
import { NotesSection } from './components/NotesSection.js';
import { useTheme } from './hooks/useTheme.js';
import { useNotes } from './hooks/useNotes.js';
import './App.css';
import { NotesGrid } from './components/NotesGrid.js';

export interface Note {
  id: string;
  url: string;
  title: string;
  summary?: string;
  date?: string;
  tags: string[];
  createdAt: string;
}

function App() {
  const { theme, toggleTheme } = useTheme();
  const {
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
  } = useNotes();

  return (
    <div className="app-container">
      <Header theme={theme} onToggleTheme={toggleTheme} />
      
      <main className="app-main">
        <NoteForm />
        
        <NotesSection
          notes={notes}
          filteredNotes={filteredNotes}
          searchKeyword={searchKeyword}
          tagFilter={tagFilter}
          loading={loading}
          onSearch={handleSearch}
          onTagFilter={handleTagFilter}
          onClearTagFilter={clearTagFilter}
          onClearSearch={clearSearch}
          onExport={handleExport}
          onImport={handleImport}
          onDelete={handleDelete}
        />

        <NotesGrid 
            notes={}
            searchKeyword={}
            onTagClick={}
            onDelete={}
            currentTagFilter={}
        />
      </main>
    </div>
  );
}

export default App;

