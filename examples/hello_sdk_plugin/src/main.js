const { definePlugin } = require('../dist/sdk/runtime');

let settingsStore = null;

function defaultSettings() {
  return {
    displayName: 'Lemon Tea',
  };
}

function normalizeSettings(raw) {
  return {
    displayName: String((raw && raw.displayName) || 'Lemon Tea').trim() || 'Lemon Tea',
  };
}

async function loadSettings() {
  return normalizeSettings(settingsStore ? settingsStore.read() : defaultSettings());
}

async function saveSettings(raw) {
  const next = normalizeSettings(raw);
  if (settingsStore) {
    settingsStore.write(next);
  }
  return next;
}

const plugin = definePlugin({
  onInitialize(ctx) {
    settingsStore = ctx.storage.jsonStore('hello-settings.json', defaultSettings);
  },
  useTools: {
    async hello_world(args) {
      const settings = await loadSettings();
      const requestedName = String((args && args.name) || '').trim();
      const name = requestedName || settings.displayName;
      return {
        ok: true,
        message: `Hello, ${name}!`,
      };
    },
  },
  viewTools: {
    async show_hello(args) {
      const settings = await loadSettings();
      return {
        viewId: 'hello_view',
        region: 'chat_side_panel',
        title: 'Hello SDK',
        data: {
          message: String((args && args.message) || `Hello, ${settings.displayName}!`),
        },
      };
    },
  },
  settings: {
    get: () => loadSettings(),
    save: (config) => saveSettings(config),
  },
});

plugin.start();
