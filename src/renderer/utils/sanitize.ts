export const escapeHtml = (text: string): string => {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
};

export const sanitizeUrl = (url: string): string => {
  if (!url || typeof url !== 'string') {
    return 'about:blank';
  }

  url = url.trim();

  const dangerousProtocols = ['javascript:', 'data:', 'vbscript:', 'file:', 'about:'];
  const lowerUrl = url.toLowerCase();

  for (const protocol of dangerousProtocols) {
    if (lowerUrl.startsWith(protocol)) {
      console.warn('Blocked dangerous URL protocol:', url);
      return 'about:blank';
    }
  }

  const allowedPatterns = [
    /^https?:\/\//i,
    /^mailto:/i,
    /^tel:/i,
    /^\//,
    /^\.\//,
    /^\.\.\//,
  ];

  const hasAllowedPattern = allowedPatterns.some((pattern) => pattern.test(url));

  if (!hasAllowedPattern) {
    const domainPattern = /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+/;
    if (domainPattern.test(url.split('/')[0].split('?')[0].split('#')[0])) {
      url = 'https://' + url;
    } else {
      console.warn('Blocked invalid URL format:', url);
      return 'about:blank';
    }
  }

  try {
    const urlObj = new URL(url, window.location.href);
    const allowedProtocols = ['http:', 'https:', 'mailto:', 'tel:'];

    if (allowedProtocols.includes(urlObj.protocol)) {
      return urlObj.href;
    } else {
      console.warn('Blocked non-allowed protocol:', urlObj.protocol);
      return 'about:blank';
    }
  } catch (e) {
    if (url.startsWith('/') || url.startsWith('./') || url.startsWith('../')) {
      return url;
    } else {
      console.warn('Failed to parse URL:', url, e);
      return 'about:blank';
    }
  }
};

