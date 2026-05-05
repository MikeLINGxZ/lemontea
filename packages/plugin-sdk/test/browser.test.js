const test = require('node:test');
const assert = require('node:assert/strict');

const { createPluginViewClient } = require('../browser');

function createMockWindow() {
  const listeners = new Map();
  return {
    addEventListener(type, handler) {
      const current = listeners.get(type) || [];
      current.push(handler);
      listeners.set(type, current);
    },
    removeEventListener(type, handler) {
      const current = listeners.get(type) || [];
      listeners.set(type, current.filter((item) => item !== handler));
    },
    emit(type, event) {
      const current = listeners.get(type) || [];
      current.forEach((handler) => handler(event));
    },
  };
}

test('browser client posts ready and resolves init payload', async () => {
  const pluginWindow = createMockWindow();
  const posted = [];
  const targetWindow = {
    postMessage(message) {
      posted.push(message);
    },
  };

  const client = createPluginViewClient({
    window: pluginWindow,
    targetWindow,
  });

  assert.equal(posted[0].type, 'lemontea-plugin-view:ready');

  const readyPromise = client.whenReady();
  pluginWindow.emit('message', {
    data: {
      type: 'lemontea-plugin-view:init',
      payload: {
        pluginId: 'demo.plugin',
        viewId: 'settings',
      },
    },
  });

  const context = await readyPromise;
  assert.equal(context.pluginId, 'demo.plugin');
  assert.equal(context.viewId, 'settings');
  client.destroy();
});

test('browser client keeps request ids stable across response resolution', async () => {
  const pluginWindow = createMockWindow();
  const posted = [];
  const targetWindow = {
    postMessage(message) {
      posted.push(message);
    },
  };

  const client = createPluginViewClient({
    window: pluginWindow,
    targetWindow,
  });

  const callPromise = client.callTool('use_tool', 'search_mail', { query: 'hello' });
  const requestEnvelope = posted[1];
  assert.equal(requestEnvelope.type, 'lemontea-plugin-view:request');
  assert.equal(requestEnvelope.payload.requestId, 'plugin-view-1');
  assert.equal(requestEnvelope.payload.payload.method, 'call_tool');

  pluginWindow.emit('message', {
    data: {
      type: 'lemontea-plugin-view:response',
      requestId: 'plugin-view-1',
      ok: true,
      result: {
        ok: true,
      },
    },
  });

  const result = await callPromise;
  assert.deepEqual(result, { ok: true });
  client.destroy();
});

test('browser client rejects host-side errors', async () => {
  const pluginWindow = createMockWindow();
  const posted = [];
  const targetWindow = {
    postMessage(message) {
      posted.push(message);
    },
  };

  const client = createPluginViewClient({
    window: pluginWindow,
    targetWindow,
  });

  const savePromise = client.saveSettings({ foo: 'bar' });
  const requestId = posted[1].payload.requestId;
  pluginWindow.emit('message', {
    data: {
      type: 'lemontea-plugin-view:response',
      requestId,
      ok: false,
      error: 'save failed',
    },
  });

  await assert.rejects(savePromise, /save failed/);
  client.destroy();
});
