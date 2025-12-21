import React, { useState, useEffect } from 'react';
import { Header } from './components/Header.js';
import { NoteForm } from './components/NoteForm.js';
import { NotesSection } from './components/NotesSection.js';
import { useTheme } from './hooks/useTheme.js';
import { useNotes } from './hooks/useNotes.js';
import './App.css';
import { NotesGrid } from './components/NotesGrid.js';
import { AppSidebar } from './components/Sidebar.js';

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
  
  const [sidebarCollapsed, setSidebarCollapsed] = useState(true);

  return (
    <div className={`app-container with-sidebar ${sidebarCollapsed ? 'sidebar-collapsed' : ''}`}>
      <Header theme={theme} onToggleTheme={toggleTheme} />
      <AppSidebar 
        onExport={handleExport}
        onImport={handleImport}
        isCollapsed={sidebarCollapsed}
        onToggleCollapse={() => setSidebarCollapsed(!sidebarCollapsed)}
      />
      
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

        {/* <NotesGrid 
            notes={notes}
            searchKeyword={searchKeyword}
            onTagClick={handleTagFilter}
            onDelete={handleDelete}
            currentTagFilter={tagFilter}
        /> */}
      </main>
    </div>
  );
}

export default App;

