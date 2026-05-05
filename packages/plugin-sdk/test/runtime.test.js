const test = require('node:test');
const assert = require('node:assert/strict');

const { definePlugin } = require('../runtime');

function createOutputBuffer() {
  const lines = [];
  return {
    lines,
    output: {
      write(chunk) {
        lines.push(JSON.parse(String(chunk).trim()));
      },
    },
  };
}

test('initialize returns manifest capabilities by default', async () => {
  const { lines, output } = createOutputBuffer();
  const runtime = definePlugin({}, { output, schedule: (fn) => fn(), exit: () => {} });

  await runtime.receive({
    id: 'req-1',
    method: 'initialize',
    params: {
      pluginId: 'demo.plugin',
      dataDir: '/tmp/demo',
      manifest: {
        capabilities: {
          useTools: [{ id: 'hello' }],
          viewTools: [{ id: 'show_hello' }],
          hooks: ['before_llm_send'],
        },
        views: [{ id: 'hello_view' }],
      },
    },
  });

  assert.equal(lines.length, 1);
  assert.deepEqual(lines[0].result.capabilities.useTools, [{ id: 'hello' }]);
  assert.deepEqual(lines[0].result.capabilities.viewTools, [{ id: 'show_hello' }]);
  assert.deepEqual(lines[0].result.capabilities.hooks, ['before_llm_send']);
  assert.deepEqual(lines[0].result.capabilities.views, [{ id: 'hello_view' }]);
});

test('call_use_tool resolves host credential round-trip', async () => {
  const { lines, output } = createOutputBuffer();
  const runtime = definePlugin({
    useTools: {
      inspect_secret: async (args, ctx) => {
        const exists = await ctx.host.credentials.exists('mail:test:imap', 'password');
        return {
          ok: true,
          seen: args.value,
          exists,
        };
      },
    },
  }, { output, schedule: (fn) => fn(), exit: () => {} });

  await runtime.receive({
    id: 'init-1',
    method: 'initialize',
    params: {
      pluginId: 'demo.plugin',
      dataDir: '/tmp/demo',
      manifest: { capabilities: {}, views: [] },
    },
  });

  const toolPromise = runtime.receive({
    id: 'tool-1',
    method: 'call_use_tool',
    params: {
      toolId: 'inspect_secret',
      args: { value: 42 },
    },
  });

  assert.equal(lines[1].method, 'get_credential');
  assert.equal(lines[1].params.scope, 'mail:test:imap');
  assert.equal(lines[1].params.key, 'password');

  await runtime.receive({
    id: lines[1].id,
    result: {
      set: true,
      value: 'secret',
    },
  });

  await toolPromise;

  assert.deepEqual(lines[2], {
    id: 'tool-1',
    protocolVersion: '1.0',
    result: {
      content: {
        ok: true,
        seen: 42,
        exists: true,
      },
    },
  });
});

test('plugin errors are returned as PLUGIN_ERROR responses', async () => {
  const { lines, output } = createOutputBuffer();
  const runtime = definePlugin({
    useTools: {
      explode() {
        throw new Error('boom');
      },
    },
  }, { output, schedule: (fn) => fn(), exit: () => {} });

  await runtime.receive({
    id: 'tool-err',
    method: 'call_use_tool',
    params: {
      toolId: 'explode',
      args: {},
    },
  });

  assert.equal(lines[0].error.code, 'PLUGIN_ERROR');
  assert.equal(lines[0].error.message, 'boom');
});

test('shutdown returns ok and exits', async () => {
  const { lines, output } = createOutputBuffer();
  const exitCalls = [];
  const runtime = definePlugin({}, {
    output,
    schedule: (fn) => fn(),
    exit: (code) => exitCalls.push(code),
  });

  await runtime.receive({
    id: 'shutdown-1',
    method: 'shutdown',
    params: {},
  });

  assert.deepEqual(lines[0], {
    id: 'shutdown-1',
    protocolVersion: '1.0',
    result: { ok: true },
  });
  assert.deepEqual(exitCalls, [0]);
});
