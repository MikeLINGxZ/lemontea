const fs = require('fs');
const path = require('path');
const readline = require('readline');

const PROTOCOL_VERSION = '1.0';

function cloneJSON(value) {
  if (value === undefined) {
    return undefined;
  }
  return JSON.parse(JSON.stringify(value));
}

function normalizeMethodMap(definition) {
  if (!definition || typeof definition !== 'object') {
    return {};
  }
  return definition;
}

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function createJSONStore(dataDir, relativePath, defaultValueFactory) {
  const filePath = path.isAbsolute(relativePath)
    ? relativePath
    : path.join(dataDir, relativePath);

  function fallbackValue() {
    const value = typeof defaultValueFactory === 'function'
      ? defaultValueFactory()
      : cloneJSON(defaultValueFactory);
    return value === undefined ? undefined : cloneJSON(value);
  }

  function read() {
    try {
      const raw = fs.readFileSync(filePath, 'utf8');
      return JSON.parse(raw);
    } catch (error) {
      return fallbackValue();
    }
  }

  function write(value) {
    ensureDir(path.dirname(filePath));
    fs.writeFileSync(filePath, JSON.stringify(value, null, 2));
    return value;
  }

  function update(updater) {
    const current = read();
    const next = updater(current);
    write(next);
    return next;
  }

  return {
    path: filePath,
    read,
    write,
    update,
  };
}

function createPluginRuntime(definition, options = {}) {
  const pluginDef = definition || {};
  const env = options.env || process.env;
  const input = options.input || process.stdin;
  const output = options.output || process.stdout;
  const exitProcess = options.exit || ((code) => process.exit(code));
  const schedule = options.schedule || ((fn) => setTimeout(fn, 10));

  const useTools = normalizeMethodMap(pluginDef.useTools);
  const viewTools = normalizeMethodMap(pluginDef.viewTools);
  const settings = normalizeMethodMap(pluginDef.settings);
  const hooks = normalizeMethodMap(pluginDef.hooks);

  let pluginId = env.LEMONTEA_PLUGIN_ID || pluginDef.pluginId || '';
  let dataDir = env.LEMONTEA_PLUGIN_DATA_DIR || path.join(process.cwd(), 'data');
  let manifest = pluginDef.manifest || null;

  let nextHostCallId = 1;
  const pendingHostCalls = new Map();

  const runtime = {
    protocolVersion: PROTOCOL_VERSION,
    get pluginId() {
      return pluginId;
    },
    get dataDir() {
      return dataDir;
    },
    get manifest() {
      return manifest;
    },
    write(message) {
      output.write(`${JSON.stringify(message)}\n`);
    },
    async callHost(method, params) {
      const id = `host-${nextHostCallId++}`;
      return new Promise((resolve, reject) => {
        pendingHostCalls.set(id, { resolve, reject });
        runtime.write({
          id,
          protocolVersion: PROTOCOL_VERSION,
          method,
          params,
        });
      });
    },
  };

  function credentialsAPI(ctx) {
    return {
      async get(scope, key) {
        const response = await runtime.callHost('get_credential', { scope, key });
        if (!response || !response.set) {
          return '';
        }
        return String(response.value || '');
      },
      async exists(scope, key) {
        const response = await runtime.callHost('get_credential', { scope, key });
        return Boolean(response && response.set);
      },
      async set(scope, key, value) {
        await runtime.callHost('set_credential', { scope, key, value });
      },
      async delete(scope, key) {
        await runtime.callHost('delete_credential', { scope, key });
      },
    };
  }

  function createContext() {
    const ctx = {
      get pluginId() {
        return pluginId;
      },
      get dataDir() {
        return dataDir;
      },
      get manifest() {
        return manifest;
      },
      env,
      host: null,
      storage: null,
      helpers: null,
      runtime,
    };

    const storage = {
      dataDir,
      path: (...segments) => path.join(dataDir, ...segments.filter(Boolean)),
      jsonStore(relativePath, defaultValue) {
        return createJSONStore(dataDir, relativePath, defaultValue);
      },
    };
    ctx.storage = storage;
    ctx.helpers = {
      jsonStore: storage.jsonStore,
    };
    ctx.host = {
      call: runtime.callHost,
      credentials: credentialsAPI(ctx),
    };
    return ctx;
  }

  function pluginError(error) {
    return {
      code: 'PLUGIN_ERROR',
      message: error instanceof Error ? error.message : String(error),
    };
  }

  function resolveCapabilities(ctx) {
    if (typeof pluginDef.capabilities === 'function') {
      return pluginDef.capabilities(ctx) || {};
    }
    return pluginDef.capabilities || {};
  }

  async function resolveInitialize(params) {
    pluginId = params.pluginId || pluginId;
    dataDir = params.dataDir || dataDir;
    manifest = params.manifest || manifest;
    ensureDir(dataDir);

    const ctx = createContext();
    if (typeof pluginDef.onInitialize === 'function') {
      await pluginDef.onInitialize(ctx, params);
    }

    const caps = resolveCapabilities(ctx);
    const mergedCapabilities = {
      useTools: caps.useTools || params.manifest?.capabilities?.useTools || [],
      viewTools: caps.viewTools || params.manifest?.capabilities?.viewTools || [],
      agents: caps.agents || params.manifest?.capabilities?.agents || [],
      views: caps.views || params.manifest?.views || [],
      hooks: caps.hooks || params.manifest?.capabilities?.hooks || [],
    };

    return {
      capabilities: mergedCapabilities,
    };
  }

  function resolveToolHandler(registry, toolId) {
    const entry = registry[String(toolId || '')];
    if (typeof entry === 'function') {
      return entry;
    }
    if (entry && typeof entry.handler === 'function') {
      return entry.handler;
    }
    return null;
  }

  async function handle(method, params = {}) {
    const ctx = createContext();

    switch (method) {
      case 'initialize':
        return resolveInitialize(params);
      case 'call_use_tool': {
        const handler = resolveToolHandler(useTools, params.toolId);
        if (!handler) {
          throw new Error(`Unknown use tool: ${params.toolId}`);
        }
        return { content: await handler(params.args || {}, ctx, params) };
      }
      case 'call_view_tool': {
        const handler = resolveToolHandler(viewTools, params.toolId);
        if (!handler) {
          throw new Error(`Unknown view tool: ${params.toolId}`);
        }
        return { content: await handler(params.args || {}, ctx, params) };
      }
      case 'get_settings': {
        const getSettings = typeof settings.get === 'function' ? settings.get : async () => ({});
        const config = await getSettings(ctx, params);
        return { config: config || {} };
      }
      case 'save_settings': {
        const saveSettings = typeof settings.save === 'function'
          ? settings.save
          : async (config) => config || {};
        const config = await saveSettings(params.config || {}, ctx, params);
        return { config: config || {} };
      }
      case 'test_connection': {
        const testConnection = typeof settings.testConnection === 'function'
          ? settings.testConnection
          : async () => ({ ok: true });
        const result = await testConnection(
          String(params.protocol || ''),
          params.config || {},
          ctx,
          params
        );
        return { result: result || {} };
      }
      case 'before_llm_send': {
        const beforeHook = typeof hooks.beforeLLMSend === 'function'
          ? hooks.beforeLLMSend
          : async (messages) => messages;
        const messages = await beforeHook(params.messages || [], ctx, params);
        return { messages: Array.isArray(messages) ? messages : (params.messages || []) };
      }
      case 'after_llm_send': {
        if (typeof hooks.afterLLMSend === 'function') {
          await hooks.afterLLMSend(params, ctx);
        }
        return { ok: true };
      }
      case 'shutdown': {
        if (typeof pluginDef.onShutdown === 'function') {
          await pluginDef.onShutdown(ctx, params);
        }
        schedule(() => exitProcess(0));
        return { ok: true };
      }
      default:
        throw new Error(`Unknown method: ${method}`);
    }
  }

  async function dispatchRequest(request) {
    try {
      const result = await handle(request.method, request.params || {});
      runtime.write({ id: request.id, protocolVersion: PROTOCOL_VERSION, result });
    } catch (error) {
      runtime.write({ id: request.id, protocolVersion: PROTOCOL_VERSION, error: pluginError(error) });
    }
  }

  async function receive(messageOrLine) {
    let message = messageOrLine;
    if (typeof messageOrLine === 'string') {
      const line = messageOrLine.trim();
      if (!line) {
        return;
      }
      try {
        message = JSON.parse(line);
      } catch (error) {
        runtime.write({ protocolVersion: PROTOCOL_VERSION, error: pluginError(error) });
        return;
      }
    }

    if (!message || typeof message !== 'object') {
      return;
    }

    if (message.method) {
      await dispatchRequest(message);
      return;
    }

    if (message.id && pendingHostCalls.has(message.id)) {
      const pending = pendingHostCalls.get(message.id);
      pendingHostCalls.delete(message.id);
      if (message.error) {
        pending.reject(new Error(message.error.message || 'Host RPC failed'));
        return;
      }
      pending.resolve(message.result || {});
    }
  }

  function start() {
    const rl = readline.createInterface({
      input,
      crlfDelay: Infinity,
    });
    rl.on('line', async (line) => {
      await receive(line);
    });
    return runtime;
  }

  runtime.receive = receive;
  runtime.start = start;
  runtime.handle = handle;
  runtime.createContext = createContext;

  return runtime;
}

function definePlugin(definition, options) {
  return createPluginRuntime(definition, options);
}

module.exports = {
  PROTOCOL_VERSION,
  createPluginRuntime,
  definePlugin,
};
