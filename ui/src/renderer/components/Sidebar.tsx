import React, { useState } from 'react';
import { MenuIcon } from './MenuIcon.js';

type NavItemProps =
  | {
      type?: 'item';
      label: string;
      icon?: string;
      onClick?: () => void;
      disabled?: boolean;
    }
  | {
      type: 'separator';
      label?: undefined;
      icon?: undefined;
      onClick?: undefined;
      disabled?: undefined;
    };

interface AppSidebarProps {
  onExport?: () => void;
  onImport?: () => void;
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
}

export function AppSidebar({ onExport, onImport, isCollapsed = true, onToggleCollapse }: AppSidebarProps) {
  const [activeItem, setActiveItem] = useState<string>('all-notes');

  const navItems: NavItemProps[] = [
    { 
      label: 'All Notes', 
      icon: '📝',
      onClick: () => {
        setActiveItem('all-notes');
        // Scroll to notes section
        document.getElementById('notes-list')?.scrollIntoView({ behavior: 'smooth' });
      }
    },
    { 
      label: 'Add Note', 
      icon: '➕',
      onClick: () => {
        setActiveItem('add-note');
        // Scroll to form
        document.querySelector('.form-section')?.scrollIntoView({ behavior: 'smooth' });
      }
    },
    { 
      label: 'Search', 
      icon: '🔍',
      onClick: () => {
        setActiveItem('search');
        // Focus search bar
        const searchInput = document.querySelector('#search-input') as HTMLInputElement;
        searchInput?.focus();
      }
    },
    { type: 'separator' },
    { 
      label: 'Export', 
      icon: '📤',
      onClick: () => {
        setActiveItem('export');
        onExport?.();
      }
    },
    { 
      label: 'Import', 
      icon: '📥',
      onClick: () => {
        setActiveItem('import');
        onImport?.();
      }
    },
  ];

  return (
    <div className={`sidebar ${isCollapsed ? 'collapsed' : ''}`}>
      <div className="sidebar-content">
        <button 
          className="sidebar-toggle"
          onClick={onToggleCollapse}
          title={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          <MenuIcon size={20} className="menu-icon" />
        </button>
        
        <div className="sidebar-nav">
          {navItems.map((item, index) => {
            if (item.type === 'separator') {
              return <div key={index} className="sidebar-separator" />;
            }

            const isActive = activeItem === item.label.toLowerCase().replace(' ', '-');
            
            return (
              <button
                key={index}
                onClick={item.onClick}
                disabled={item.disabled}
                className={`sidebar-item ${isActive ? 'active' : ''} ${item.disabled ? 'disabled' : ''}`}
                title={isCollapsed ? item.label : undefined}
              >
                {item.icon && <span className="sidebar-icon">{item.icon}</span>}
                {!isCollapsed && <span className="sidebar-label">{item.label}</span>}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
