import { beforeEach, describe, expect, it, vi } from 'vitest';
import { openConnectionWindow } from './openConnectionWindow';

describe('openConnectionWindow', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.stubGlobal('crypto', {
      randomUUID: vi.fn()
        .mockReturnValueOnce('uuid-1')
        .mockReturnValueOnce('uuid-2'),
    });
    Object.defineProperty(window, 'screen', {
      value: { width: 1920, height: 1080 },
      configurable: true,
    });
  });

  it('opens same-connection popups with unique tab ids and window names', () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);

    openConnectionWindow('conn-1');
    openConnectionWindow('conn-1');

    expect(openSpy).toHaveBeenNthCalledWith(
      1,
      '/connection/conn-1?tabId=popup-conn-1-uuid-1',
      'arsenale-popup-conn-1-uuid-1',
      'width=1024,height=768,left=448,top=156,menubar=no,toolbar=no,location=no,status=no',
    );
    expect(openSpy).toHaveBeenNthCalledWith(
      2,
      '/connection/conn-1?tabId=popup-conn-1-uuid-2',
      'arsenale-popup-conn-1-uuid-2',
      'width=1024,height=768,left=448,top=156,menubar=no,toolbar=no,location=no,status=no',
    );
  });
});
