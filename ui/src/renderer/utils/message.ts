export const showMessage = (message: string, type: 'success' | 'error' | 'info' = 'info') => {
  let messageElement = document.getElementById('message');
  
  if (!messageElement) {
    messageElement = document.createElement('div');
    messageElement.id = 'message';
    messageElement.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      padding: 12px 20px;
      border-radius: 8px;
      color: white;
      font-weight: 600;
      z-index: 1000;
      transition: all 0.3s ease;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
    `;
    document.body.appendChild(messageElement);
  }

  messageElement.textContent = message;
  const rootStyles = getComputedStyle(document.documentElement);
  messageElement.style.backgroundColor =
    type === 'success'
      ? rootStyles.getPropertyValue('--success-color').trim()
      : type === 'error'
      ? rootStyles.getPropertyValue('--error-color').trim()
      : rootStyles.getPropertyValue('--info-color').trim();

  messageElement.style.opacity = '1';
  messageElement.style.transform = 'translateY(0)';

  setTimeout(() => {
    messageElement!.style.opacity = '0';
    messageElement!.style.transform = 'translateY(-20px)';
  }, 3000);
};

