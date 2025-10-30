# Fonso - URL Note Taker

A simple Electron desktop application for saving and managing interesting URLs with metadata.

## Features

- Save URLs with title, summary, and date
- View all saved notes in a clean interface
- Delete notes you no longer need
- Persistent storage using JSON files
- Modern, responsive UI

## Getting Started

### Prerequisites

- Node.js (version 14 or higher)
- npm

### Installation

1. Clone this repository
2. Install dependencies:
   ```bash
   npm install
   ```

### Running the App

```bash
npm start
```

## Usage

1. **Adding a Note**: Fill in the form on the left side:
   - URL (required): The web address you want to save
   - Title (required): A descriptive title for the URL
   - Summary (optional): Why you saved this URL or what's interesting about it
   - Date: Automatically set to current date/time

2. **Viewing Notes**: All your saved notes appear on the right side, sorted by creation date (newest first)

3. **Deleting Notes**: Click the trash icon (🗑️) on any note to delete it

## Data Storage

Notes are stored in a `data/notes.json` file in the application directory. This file is created automatically when you save your first note.

## Development

This app was built as a learning project to understand Electron development. It demonstrates:

- Main process vs Renderer process architecture
- IPC (Inter-Process Communication) between processes
- File system access in Electron
- Modern JavaScript (ES6+) features
- Responsive web design
- Security best practices

## Future Enhancements

- Keyword search functionality
- Semantic/context search
- Tags or categories
- Export/import features
- Rich text editing
- Dark mode theme

## License

MIT