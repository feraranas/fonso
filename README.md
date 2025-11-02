# Fonso - URL Note Taker

A modern Electron desktop app for saving and organizing URLs with metadata, built with React, TypeScript, and the Grid.js library.

## Features

✨ **Core Features:**
- 📝 Save URLs with title, summary, date, and tags
- 🔍 Full-text search across all notes
- 🏷️ Tag-based filtering
- 📤 Export notes to JSON
- 📥 Import notes from JSON
- 🌓 Light/Dark theme toggle
- 🗑️ Delete notes

🎯 **DataGrid View:**
- Sortable columns (Title, URL, Created Date)
- Resizable columns
- Interactive tag filtering
- Clean, modern interface

## Tech Stack

- **Electron** - Desktop app framework
- **React 19** - UI library
- **TypeScript** - Type safety
- **Vite** - Fast build tool & dev server
- **Grid.js** - Advanced table plugin
- **Bun** - Fast package manager and runtime

## Prerequisites

- [Bun](https://bun.sh) (version 1.0 or higher)
- macOS, Linux, or Windows

## Installation

1. **Clone the repository:**
   ```bash
   git clone <your-repo-url>
   cd fonso
   ```

2. **Install dependencies:**
   ```bash
   bun install
   ```

## Running the App

### Development Mode (with hot reload)

```bash
bun run dev
```

This starts both Vite dev server and Electron with hot module replacement enabled.

### Production Mode

1. **Build the app:**
   ```bash
   bun run build
   ```

2. **Run the production build:**
   ```bash
   bun run start
   ```

## Project Structure

```
fonso/
├── src/
│   ├── renderer/           # React frontend
│   │   ├── components/     # React components
│   │   ├── hooks/          # Custom React hooks
│   │   ├── utils/          # Utility functions
│   │   ├── App.tsx         # Main app component
│   │   └── main.tsx        # React entry point
│   └── client.ts           # MCP client (optional)
├── data/                   # Local data storage
│   └── notes.json          # Notes database
├── main.js                 # Electron main process
├── preload.js              # Electron preload script
├── index.html              # HTML entry point
├── styles.css              # Global styles
├── vite.config.ts          # Vite configuration
└── package.json            # Project dependencies

```

## Available Scripts

| Command | Description |
|---------|-------------|
| `bun run dev` | Start development mode with hot reload |
| `bun run build` | Build the React app for production |
| `bun run start` | Run Electron with production build |
| `bun run build:ts` | Compile TypeScript files |
| `bun run mcp-client` | Run MCP client (if configured) |

## Data Storage

Notes are stored locally in `data/notes.json` as a JSON array. The file is automatically created on first run.

### Note Format

```json
{
  "id": "unique-id",
  "url": "https://example.com",
  "title": "Example Site",
  "summary": "Optional description",
  "date": "2024-01-01T12:00:00Z",
  "tags": ["tag1", "tag2"],
  "createdAt": "2024-01-01T12:00:00.000Z",
  "updatedAt": "2024-01-01T12:00:00.000Z"
}
```

## Security

- **Context Isolation** enabled
- **Node Integration** disabled in renderer
- **Content Security Policy** enforced
- **URL sanitization** to prevent XSS
- **Safe preload script** for IPC communication

## Development

### Hot Reload

Changes to React components are instantly reflected without restarting Electron thanks to Vite's HMR.

### TypeScript

The project uses TypeScript for type safety. Type definitions are in `src/renderer/types.d.ts`.

### Debugging

Dev tools are automatically opened in development mode. Check the Console tab for logs.

## Migration Notes

This app was migrated from vanilla JavaScript to React. See [REACT_MIGRATION.md](REACT_MIGRATION.md) for details about the migration process and architecture decisions.

## Contributing

Feel free to open issues or submit pull requests!

## License

MIT

---

**Made with ❤️ using React + Electron + Bun**
