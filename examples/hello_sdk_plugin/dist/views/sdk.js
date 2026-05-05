(function bootstrapPluginViewSDK(globalScope) {
  function createPluginViewClient(options) {
    const currentWindow = options && options.window
      ? options.window
      : globalScope.window;
    const targetWindow = options && options.targetWindow
      ? options.targetWindow
      : (currentWindow && currentWindow.parent && currentWindow.parent !== currentWindow
        ? currentWindow.parent
        : null);
    const targetOrigin = options && options.targetOrigin ? options.targetOrigin : '*';
    const autoReady = !options || options.autoReady !== false;

    let initPayload = null;
    let nextRequestId = 1;
    const pending = new Map();
    const initWaiters = [];

    function flushInitWaiters() {
      while (initWaiters.length > 0) {
        const resolve = initWaiters.shift();
        resolve(initPayload);
      }
    }

    function post(type, payload) {
      if (!targetWindow || typeof targetWindow.postMessage !== 'function') {
        throw new Error('Plugin views must be opened inside Lemon Tea.');
      }
      targetWindow.postMessage({ type, payload }, targetOrigin);
    }

    function request(method, params) {
      const requestId = `plugin-view-${nextRequestId++}`;
      return new Promise((resolve, reject) => {
        pending.set(requestId, { resolve, reject });
        post('lemontea-plugin-view:request', {
          requestId,
          payload: { method, params: params || {} },
        });
      });
    }

    function handleMessage(event) {
      const data = event && event.data ? event.data : {};
      if (data.type === 'lemontea-plugin-view:init') {
        initPayload = data.payload || null;
        flushInitWaiters();
        return;
      }
      if (data.type !== 'lemontea-plugin-view:response') {
        return;
      }
      const requestId = String(data.requestId || '');
      const current = pending.get(requestId);
      if (!current) {
        return;
      }
      pending.delete(requestId);
      if (data.ok) {
        current.resolve(data.result);
        return;
      }
      current.reject(new Error(String(data.error || 'Plugin view request failed.')));
    }

    if (currentWindow && typeof currentWindow.addEventListener === 'function') {
      currentWindow.addEventListener('message', handleMessage);
    }

    const client = {
      whenReady() {
        if (initPayload) {
          return Promise.resolve(initPayload);
        }
        return new Promise((resolve) => {
          initWaiters.push(resolve);
        });
      },
      request,
      getContext() {
        return request('get_context');
      },
      callTool(kind, toolId, args) {
        return request('call_tool', { kind, toolId, args });
      },
      getSettings() {
        return request('get_settings');
      },
      saveSettings(config) {
        return request('save_settings', { config });
      },
      testConnection(protocol, config) {
        return request('test_connection', { protocol, config });
      },
      updateView(payload) {
        return request('update_view', { payload });
      },
      openView(payload) {
        return request('open_view', { payload });
      },
      composeMessage(text) {
        return request('compose_message', { text });
      },
      destroy() {
        if (currentWindow && typeof currentWindow.removeEventListener === 'function') {
          currentWindow.removeEventListener('message', handleMessage);
        }
      },
    };

    if (autoReady) {
      post('lemontea-plugin-view:ready', {});
    }

    return client;
  }

  if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
      createPluginViewClient,
    };
  }

  if (globalScope && globalScope.window) {
    globalScope.createPluginViewClient = createPluginViewClient;
    if (!globalScope.LemonTeaPluginView) {
      globalScope.LemonTeaPluginView = createPluginViewClient();
    }
  }
})(typeof globalThis !== 'undefined' ? globalThis : this);
