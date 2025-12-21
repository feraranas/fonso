import React from 'react';

interface HeaderProps {
  theme: string;
  onToggleTheme: () => void;
}

export const Header: React.FC<HeaderProps> = ({ theme, onToggleTheme }) => {
  return (
    <header className="app-header">
      <button
        id="theme-toggle"
        className="theme-toggle"
        title="Toggle dark mode"
        onClick={onToggleTheme}
      >
        <span className="theme-icon">{theme === 'dark' ? '☀️' : '🌙'}</span>
      </button>
      <h5>📝 Fonso - URL Note Taker</h5>
    </header>
  );
};

