/**
 * Keyboard Lock API (Chromium-only).
 * Not included in TypeScript's standard lib.dom.d.ts.
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Keyboard_API
 */

interface Keyboard {
  lock(keyCodes?: string[]): Promise<void>;
  unlock(): void;
}

interface Navigator {
  readonly keyboard?: Keyboard;
}
