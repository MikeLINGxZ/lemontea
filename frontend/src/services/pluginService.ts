import { Service } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service';
import type { PluginAgent, PluginTool, PluginView, RuntimeStatus, Summary } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/plugins';

export type { PluginAgent, PluginTool, PluginView, RuntimeStatus };
export type PluginSummary = Summary;

export function getPluginRuntimeStatus(): Promise<RuntimeStatus> {
  return Service.GetPluginRuntimeStatus();
}

export function downloadPluginRuntime(): Promise<RuntimeStatus | null> {
  return Service.DownloadPluginRuntime();
}

export function listPlugins(): Promise<PluginSummary[]> {
  return Service.ListPlugins().then((res) => res || []);
}

export function getPlugin(id: string): Promise<PluginSummary | null> {
  return Service.GetPlugin(id);
}

export function selectPluginFolder(): Promise<string> {
  return Service.SelectPluginFolder();
}

export function addPluginFromFolder(path: string): Promise<PluginSummary | null> {
  return Service.AddPluginFromFolder(path);
}

export function setPluginEnabled(id: string, enabled: boolean): Promise<void> {
  return Service.SetPluginEnabled(id, enabled);
}

export function deletePlugin(id: string): Promise<void> {
  return Service.DeletePlugin(id);
}

export function openPluginSettingsWindow(id: string): Promise<void> {
  return Service.OpenPluginSettingsWindow(id);
}
