import { buildConnectionViewerUrl, createTabInstanceId } from './tabInstance';

export function openConnectionWindow(connectionId: string) {
  const width = 1024;
  const height = 768;
  const left = Math.round((window.screen.width - width) / 2);
  const top = Math.round((window.screen.height - height) / 2);
  const tabId = createTabInstanceId('popup', connectionId);

  window.open(
    buildConnectionViewerUrl(connectionId, tabId),
    `arsenale-${tabId}`,
    `width=${width},height=${height},left=${left},top=${top},menubar=no,toolbar=no,location=no,status=no`
  );
}
