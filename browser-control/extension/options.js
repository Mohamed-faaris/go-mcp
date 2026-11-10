const url = document.getElementById('url');
const token = document.getElementById('token');
const saved = document.getElementById('saved');
chrome.storage.local.get({ bridgeUrl: 'ws://127.0.0.1:8765/bridge', token: '' }, value => {
  url.value = value.bridgeUrl;
  token.value = value.token;
});
document.getElementById('save').addEventListener('click', async () => {
  const value = token.value.trim();
  if (value.length < 32) { saved.textContent = 'Token must contain at least 32 characters.'; return; }
  await chrome.storage.local.set({ bridgeUrl: url.value.trim(), token: value });
  saved.textContent = 'Saved. The extension is reconnecting.';
});