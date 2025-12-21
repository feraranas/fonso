import React, { useState, useEffect } from 'react';
import { showMessage } from '../utils/message.js';

interface NoteFormData {
  url: string;
  title: string;
  summary: string;
  date: string;
  tags: string;
}

export const NoteForm: React.FC = () => {
  const [formData, setFormData] = useState<NoteFormData>({
    url: '',
    title: '',
    summary: '',
    date: '',
    tags: '',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    // Set current date/time as default
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    const hours = String(now.getHours()).padStart(2, '0');
    const minutes = String(now.getMinutes()).padStart(2, '0');
    const formattedDate = `${year}-${month}-${day}T${hours}:${minutes}`;
    setFormData((prev) => ({ ...prev, date: formattedDate }));
  }, []);

  const parseTags = (tagString: string): string[] => {
    if (!tagString || !tagString.trim()) {
      return [];
    }
    return tagString
      .split(',')
      .map((tag) => tag.trim().toLowerCase())
      .filter((tag) => tag.length > 0);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!formData.url || !formData.title) {
      showMessage('Please fill in the required fields (URL and Title)', 'error');
      return;
    }

    setIsSubmitting(true);

    try {
      const noteData = {
        url: formData.url,
        title: formData.title,
        summary: formData.summary,
        date: formData.date,
        tags: parseTags(formData.tags),
      };

      const result = await window.electronAPI.saveNote(noteData);

      if (result.success) {
        showMessage('Note saved successfully!', 'success');
        // Reset form
        setFormData({
          url: '',
          title: '',
          summary: '',
          date: '',
          tags: '',
        });
        // Reset date to current time
        const now = new Date();
        const year = now.getFullYear();
        const month = String(now.getMonth() + 1).padStart(2, '0');
        const day = String(now.getDate()).padStart(2, '0');
        const hours = String(now.getHours()).padStart(2, '0');
        const minutes = String(now.getMinutes()).padStart(2, '0');
        const formattedDate = `${year}-${month}-${day}T${hours}:${minutes}`;
        setFormData((prev) => ({ ...prev, date: formattedDate }));
      } else {
        showMessage(`Error saving note: ${result.error}`, 'error');
      }
    } catch (error) {
      console.error('Error saving note:', error);
      showMessage('Error saving note. Please try again.', 'error');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  return (
    <div className="form-section">
      <h2>Add New URL Note</h2>
      <form onSubmit={handleSubmit} className="note-form">
        <div className="form-group">
          <label htmlFor="url">URL *</label>
          <input
            type="url"
            id="url"
            name="url"
            placeholder="https://example.com"
            required
            className="form-input"
            value={formData.url}
            onChange={handleChange}
          />
        </div>

        <div className="form-group">
          <label htmlFor="title">Title *</label>
          <input
            type="text"
            id="title"
            name="title"
            placeholder="Enter a descriptive title"
            required
            className="form-input"
            value={formData.title}
            onChange={handleChange}
          />
        </div>

        <div className="form-group">
          <label htmlFor="summary">Summary</label>
          <textarea
            id="summary"
            name="summary"
            placeholder="What's interesting about this URL? Why did you save it?"
            rows={4}
            className="form-textarea"
            value={formData.summary}
            onChange={handleChange}
          />
        </div>

        <div className="form-group">
          <label htmlFor="date">Date</label>
          <input
            type="datetime-local"
            id="date"
            name="date"
            className="form-input"
            value={formData.date}
            onChange={handleChange}
          />
        </div>

        <div className="form-group">
          <label htmlFor="tags">Tags</label>
          <input
            type="text"
            id="tags"
            name="tags"
            placeholder="javascript, tutorial, beginner (comma-separated)"
            className="form-input"
            value={formData.tags}
            onChange={handleChange}
          />
          <small className="form-hint">Separate multiple tags with commas</small>
        </div>

        <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
          {isSubmitting ? '💾 Saving...' : '💾 Save Note'}
        </button>
      </form>
    </div>
  );
};

