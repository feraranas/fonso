import React from 'react';

interface SearchBarProps {
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
}

export const SearchBar: React.FC<SearchBarProps> = ({ value, onChange, onClear }) => {
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClear();
    }
  };

  return (
    <div className="search-container">
      <div className="search-input-wrapper">
        <span className="search-icon">🔍</span>
        <input
          type="text"
          id="search-input"
          className="search-input"
          placeholder="Search notes by keyword..."
          autoComplete="off"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
        />
        {value && (
          <button
            id="clear-search-btn"
            className="btn-clear-search"
            title="Clear search"
            onClick={onClear}
          >
            ✕
          </button>
        )}
      </div>
    </div>
  );
};

