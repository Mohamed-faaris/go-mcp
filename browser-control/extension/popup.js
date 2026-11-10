const status = document.getElementById('status');
chrome.storage.local.get({ token: '', bridgeUrl: 'ws://127.0.0.1:8765/bridge' }, value => {
  status.textContent = value.token ? 'Bridge configured. Check the extension badge for connection status.' : 'Not configured. Add the shared token in settings.';
});
document.getElementById('settings').addEventListener('click', () => chrome.runtime.openOptionsPage());