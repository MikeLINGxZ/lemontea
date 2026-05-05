package plugins

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/logger"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/utils"
)

const pluginNodeVersion = "v22.22.2"

type Manager struct {
	mu              sync.RWMutex
	downloadMu      sync.RWMutex
	dataPath        string
	runtimePath     string
	pluginsPath     string
	statePath       string
	logsPath        string
	credentialStore *credentialStore
	records         map[string]*PluginRecord
	processes       map[string]*pluginProcess
	downloadStatus  RuntimeStatus
}

type pluginProcess struct {
	cmd    *exec.Cmd
	rpc    *rpcClient
	cancel context.CancelFunc
	log    *os.File
}

func NewManager() (*Manager, error) {
	dataPath, err := utils.GetDataPath()
	if err != nil {
		return nil, err
	}
	m := &Manager{
		dataPath:    dataPath,
		runtimePath: filepath.Join(dataPath, "plugin_runtime"),
		pluginsPath: filepath.Join(dataPath, "plugins", "installed"),
		statePath:   filepath.Join(dataPath, "plugins", "state", "plugins.json"),
		logsPath:    filepath.Join(dataPath, "plugins", "logs"),
		records:     map[string]*PluginRecord{},
		processes:   map[string]*pluginProcess{},
	}
	m.credentialStore = newCredentialStore(filepath.Dir(m.statePath))
	if err := m.ensureDirs(); err != nil {
		return nil, err
	}
	if err := m.loadState(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) ensureDirs() error {
	for _, dir := range []string{
		m.runtimePath,
		m.pluginsPath,
		filepath.Dir(m.statePath),
		m.logsPath,
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) RuntimePath() string {
	return m.runtimePath
}

func (m *Manager) RuntimeAvailable() bool {
	_, err := m.nodePath()
	return err == nil
}

func (m *Manager) RuntimeStatus(ctx context.Context) RuntimeStatus {
	nodePath, err := m.nodePath()
	status := RuntimeStatus{
		Available:   err == nil,
		RuntimePath: m.runtimePath,
		NodePath:    nodePath,
		Version:     pluginNodeVersion,
	}
	if url, urlErr := nodeDownloadURL(); urlErr == nil {
		status.DownloadURL = url
	} else if err == nil {
		status.Error = urlErr.Error()
	}
	if err != nil {
		status.Error = err.Error()
		return m.mergeRuntimeDownloadStatus(status)
	}
	versionCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, versionErr := exec.CommandContext(versionCtx, nodePath, "--version").Output()
	if versionErr != nil {
		status.Available = false
		status.Error = versionErr.Error()
		return m.mergeRuntimeDownloadStatus(status)
	}
	status.Version = strings.TrimSpace(string(out))
	return m.mergeRuntimeDownloadStatus(status)
}

func (m *Manager) DownloadRuntime(ctx context.Context) (RuntimeStatus, error) {
	url, err := nodeDownloadURL()
	if err != nil {
		return m.RuntimeStatus(ctx), err
	}
	if !m.beginRuntimeDownload(url) {
		return m.RuntimeStatus(ctx), fmt.Errorf("plugin runtime download is already in progress")
	}
	if err := os.MkdirAll(m.runtimePath, 0755); err != nil {
		m.finishRuntimeDownload(ctx, "failed", err)
		return m.RuntimeStatus(ctx), err
	}

	archivePath := filepath.Join(m.runtimePath, filepath.Base(url))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		m.finishRuntimeDownload(ctx, "failed", err)
		return m.RuntimeStatus(ctx), err
	}
	m.updateRuntimeDownload("downloading", 0, 0, true, "")
	resp, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
	if err != nil {
		m.finishRuntimeDownload(ctx, "failed", err)
		return m.RuntimeStatus(ctx), err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("download node runtime failed: %s", resp.Status)
		m.finishRuntimeDownload(ctx, "failed", err)
		return m.RuntimeStatus(ctx), err
	}

	out, err := os.Create(archivePath)
	if err != nil {
		m.finishRuntimeDownload(ctx, "failed", err)
		return m.RuntimeStatus(ctx), err
	}
	reader := &runtimeDownloadReader{
		reader: resp.Body,
		total:  resp.ContentLength,
		onProgress: func(downloaded, total int64) {
			m.updateRuntimeDownload("downloading", downloaded, total, true, "")
		},
	}
	if _, err = io.Copy(out, reader); err != nil {
		_ = out.Close()
		m.finishRuntimeDownload(ctx, "failed", err)
		return m.RuntimeStatus(ctx), err
	}
	if err = out.Close(); err != nil {
		m.finishRuntimeDownload(ctx, "failed", err)
		return m.RuntimeStatus(ctx), err
	}
	defer os.Remove(archivePath)

	extractPath := filepath.Join(m.runtimePath, "download")
	_ = os.RemoveAll(extractPath)
	if err := os.MkdirAll(extractPath, 0755); err != nil {
		m.finishRuntimeDownload(ctx, "failed", err)
		return m.RuntimeStatus(ctx), err
	}
	defer os.RemoveAll(extractPath)

	m.updateRuntimeDownload("extracting", reader.downloaded, reader.total, true, "")
	if strings.HasSuffix(archivePath, ".zip") {
		err = extractZip(archivePath, extractPath)
	} else {
		err = extractTarGz(archivePath, extractPath)
	}
	if err != nil {
		m.finishRuntimeDownload(ctx, "failed", err)
		return m.RuntimeStatus(ctx), err
	}

	root, err := firstExtractedDir(extractPath)
	if err != nil {
		m.finishRuntimeDownload(ctx, "failed", err)
		return m.RuntimeStatus(ctx), err
	}
	target := filepath.Join(m.runtimePath, "node")
	_ = os.RemoveAll(target)
	m.updateRuntimeDownload("installing", reader.downloaded, reader.total, true, "")
	if err := os.Rename(root, target); err != nil {
		m.finishRuntimeDownload(ctx, "failed", err)
		return m.RuntimeStatus(ctx), err
	}

	status := m.RuntimeStatus(ctx)
	if !status.Available {
		err := fmt.Errorf("%s", status.Error)
		m.finishRuntimeDownload(ctx, "failed", err)
		return status, err
	}
	m.finishRuntimeDownload(ctx, "ready", nil)
	status = m.RuntimeStatus(ctx)
	return status, nil
}

func (m *Manager) beginRuntimeDownload(url string) bool {
	m.downloadMu.Lock()
	defer m.downloadMu.Unlock()
	if m.downloadStatus.Downloading {
		return false
	}
	m.downloadStatus = RuntimeStatus{
		Available:   false,
		Version:     pluginNodeVersion,
		DownloadURL: url,
		Downloading: true,
		Progress:    0,
		Phase:       "preparing",
	}
	return true
}

func (m *Manager) updateRuntimeDownload(phase string, downloaded, total int64, downloading bool, errMessage string) {
	progress := 0
	if phase == "extracting" {
		progress = 90
	} else if phase == "installing" {
		progress = 96
	} else if phase == "ready" {
		progress = 100
	} else if total > 0 {
		progress = int(downloaded * 100 / total)
		if progress > 89 {
			progress = 89
		}
	}
	if downloaded < 0 {
		downloaded = 0
	}
	m.downloadMu.Lock()
	defer m.downloadMu.Unlock()
	m.downloadStatus.Downloading = downloading
	m.downloadStatus.Progress = progress
	m.downloadStatus.DownloadedBytes = downloaded
	m.downloadStatus.TotalBytes = total
	m.downloadStatus.Phase = phase
	m.downloadStatus.Error = errMessage
}

func (m *Manager) finishRuntimeDownload(ctx context.Context, phase string, err error) {
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}
	m.updateRuntimeDownload(phase, 0, 0, false, errMessage)
	if err == nil {
		status := m.RuntimeStatus(ctx)
		m.downloadMu.Lock()
		m.downloadStatus.Available = status.Available
		m.downloadStatus.NodePath = status.NodePath
		m.downloadStatus.Version = status.Version
		m.downloadStatus.Progress = 100
		m.downloadStatus.Phase = phase
		m.downloadMu.Unlock()
	}
}

func (m *Manager) mergeRuntimeDownloadStatus(status RuntimeStatus) RuntimeStatus {
	m.downloadMu.RLock()
	download := m.downloadStatus
	m.downloadMu.RUnlock()
	if !download.Downloading && download.Phase == "" {
		return status
	}
	status.Downloading = download.Downloading
	status.Progress = download.Progress
	status.DownloadedBytes = download.DownloadedBytes
	status.TotalBytes = download.TotalBytes
	status.Phase = download.Phase
	if download.DownloadURL != "" {
		status.DownloadURL = download.DownloadURL
	}
	if download.Error != "" {
		status.Error = download.Error
	}
	return status
}

type runtimeDownloadReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	lastUpdate time.Time
	onProgress func(downloaded, total int64)
}

func (r *runtimeDownloadReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.downloaded += int64(n)
		if time.Since(r.lastUpdate) > 200*time.Millisecond || r.downloaded == r.total {
			r.lastUpdate = time.Now()
			r.onProgress(r.downloaded, r.total)
		}
	}
	return n, err
}

func (m *Manager) nodePath() (string, error) {
	candidates := []string{
		filepath.Join(m.runtimePath, "node", "bin", "node"),
		filepath.Join(m.runtimePath, "node", "node.exe"),
		filepath.Join(m.runtimePath, "bin", "node"),
		filepath.Join(m.runtimePath, "node"),
		filepath.Join(m.runtimePath, "node.exe"),
	}
	for _, candidate := range candidates {
		if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("node runtime not found in %s", m.runtimePath)
}

func (m *Manager) loadState() error {
	raw, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var records map[string]*PluginRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		return err
	}
	if records == nil {
		records = map[string]*PluginRecord{}
	}
	m.records = records
	return nil
}

func (m *Manager) saveStateLocked() error {
	raw, err := json.MarshalIndent(m.records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.statePath, raw, 0644)
}

func (m *Manager) StartEnabled(ctx context.Context) {
	m.mu.RLock()
	ids := make([]string, 0, len(m.records))
	for id, record := range m.records {
		if record.Enabled {
			ids = append(ids, id)
		}
	}
	m.mu.RUnlock()
	for _, id := range ids {
		if err := m.Enable(ctx, id); err != nil {
			logger.Warm("start plugin failed:", id, err)
		}
	}
}

func (m *Manager) List() []Summary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]Summary, 0, len(m.records))
	for _, record := range m.records {
		res = append(res, recordToSummary(record))
	}
	sort.Slice(res, func(i, j int) bool {
		return strings.ToLower(res[i].Name) < strings.ToLower(res[j].Name)
	})
	return res
}

func (m *Manager) Get(id string) (*Summary, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	record, ok := m.records[id]
	if !ok {
		return nil, false
	}
	summary := recordToSummary(record)
	return &summary, true
}

func (m *Manager) SettingsURL(id string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	record, ok := m.records[id]
	if !ok {
		return "", fmt.Errorf("plugin %s not found", id)
	}
	if record.Manifest.SettingsView == nil || strings.TrimSpace(record.Manifest.SettingsView.Entry) == "" {
		return "", fmt.Errorf("plugin %s does not provide settings", id)
	}
	entry := strings.TrimSpace(record.Manifest.SettingsView.Entry)
	if strings.HasPrefix(entry, "http://") || strings.HasPrefix(entry, "https://") || strings.HasPrefix(entry, "/") {
		if strings.Contains(entry, "?") {
			return entry + "&id=" + id, nil
		}
		return entry + "?id=" + id, nil
	}
	return "file://" + filepath.ToSlash(filepath.Join(record.InstallPath, entry)), nil
}

func (m *Manager) InstallFromFolder(path string) (*Summary, error) {
	manifest, err := readManifest(path)
	if err != nil {
		return nil, err
	}
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}
	installPath := filepath.Join(m.pluginsPath, manifest.ID)
	dataPath := filepath.Join(installPath, "data")

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.records[manifest.ID]; exists {
		return nil, fmt.Errorf("plugin %s already installed", manifest.ID)
	}
	if err := copyDir(path, installPath); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, err
	}
	record := &PluginRecord{
		Manifest:       *manifest,
		InstallPath:    installPath,
		DataPath:       dataPath,
		Enabled:        false,
		Status:         StatusDisabled,
		Runtime:        mergeCapabilities(manifest),
		InstalledAt:    time.Now().Format(time.RFC3339),
		RuntimeHealthy: m.RuntimeAvailable(),
	}
	m.records[manifest.ID] = record
	if err := m.saveStateLocked(); err != nil {
		return nil, err
	}
	summary := recordToSummary(record)
	return &summary, nil
}

func (m *Manager) Enable(ctx context.Context, id string) error {
	m.mu.Lock()
	record, ok := m.records[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("plugin %s not found", id)
	}
	if _, running := m.processes[id]; running {
		record.Enabled = true
		record.Status = StatusEnabled
		err := m.saveStateLocked()
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()

	if err := m.startProcess(ctx, record); err != nil {
		m.mu.Lock()
		record.Enabled = false
		record.Status = StatusError
		record.LastError = err.Error()
		_ = m.saveStateLocked()
		m.mu.Unlock()
		return err
	}

	m.mu.Lock()
	record.Enabled = true
	record.Status = StatusEnabled
	record.LastError = ""
	record.LastStartedAt = time.Now().Format(time.RFC3339)
	record.RuntimeHealthy = true
	err := m.saveStateLocked()
	m.mu.Unlock()
	return err
}

func (m *Manager) Disable(id string) error {
	m.stopProcess(id)
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.records[id]
	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}
	record.Enabled = false
	record.Status = StatusDisabled
	record.LastStoppedAt = time.Now().Format(time.RFC3339)
	return m.saveStateLocked()
}

func (m *Manager) Delete(id string) error {
	_ = m.Disable(id)
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.records[id]
	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}
	delete(m.records, id)
	if err := os.RemoveAll(record.InstallPath); err != nil {
		return err
	}
	return m.saveStateLocked()
}

func (m *Manager) GetSettings(ctx context.Context, id string) (map[string]interface{}, error) {
	var result struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := m.callPluginRPC(ctx, id, "get_settings", map[string]any{}, 30*time.Second, &result); err != nil {
		return nil, err
	}
	if result.Config == nil {
		result.Config = map[string]interface{}{}
	}
	return result.Config, nil
}

func (m *Manager) SaveSettings(ctx context.Context, id string, config map[string]interface{}) (map[string]interface{}, error) {
	var result struct {
		Config map[string]interface{} `json:"config"`
	}
	params := map[string]any{
		"config": config,
	}
	if err := m.callPluginRPC(ctx, id, "save_settings", params, 30*time.Second, &result); err != nil {
		return nil, err
	}
	if result.Config == nil {
		result.Config = map[string]interface{}{}
	}
	return result.Config, nil
}

func (m *Manager) TestConnection(ctx context.Context, id, protocol string, config map[string]interface{}) (map[string]interface{}, error) {
	var result struct {
		Result map[string]interface{} `json:"result"`
	}
	params := map[string]any{
		"protocol": protocol,
		"config":   config,
	}
	if err := m.callPluginRPC(ctx, id, "test_connection", params, 45*time.Second, &result); err != nil {
		return nil, err
	}
	if result.Result == nil {
		result.Result = map[string]interface{}{}
	}
	return result.Result, nil
}

func (m *Manager) startProcess(ctx context.Context, record *PluginRecord) error {
	node, err := m.nodePath()
	if err != nil {
		return err
	}
	mainPath := filepath.Join(record.InstallPath, record.Manifest.Main)
	if _, err := os.Stat(mainPath); err != nil {
		return err
	}
	procCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(procCtx, node, mainPath)
	cmd.Dir = record.InstallPath
	cmd.Env = append(os.Environ(),
		"LEMONTEA_PLUGIN_ID="+record.Manifest.ID,
		"LEMONTEA_PLUGIN_DATA_DIR="+record.DataPath,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return err
	}
	logFile, err := os.OpenFile(filepath.Join(m.logsPath, record.Manifest.ID+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		cancel()
		return err
	}
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		cancel()
		_ = logFile.Close()
		return err
	}
	rpc := newRPCClient(record.Manifest.ID, stdin, stdout, m.pluginHostRPCHandler(record.Manifest.ID))
	process := &pluginProcess{cmd: cmd, rpc: rpc, cancel: cancel, log: logFile}

	m.mu.Lock()
	m.processes[record.Manifest.ID] = process
	m.mu.Unlock()

	go func() {
		err := cmd.Wait()
		rpc.close()
		_ = logFile.Close()
		m.mu.Lock()
		if current := m.processes[record.Manifest.ID]; current == process {
			delete(m.processes, record.Manifest.ID)
			if rec := m.records[record.Manifest.ID]; rec != nil && rec.Enabled {
				rec.Status = StatusError
				if err != nil {
					rec.LastError = err.Error()
				} else {
					rec.LastError = "plugin process exited"
				}
				_ = m.saveStateLocked()
			}
		}
		m.mu.Unlock()
	}()

	var initRes struct {
		Capabilities *Capabilities `json:"capabilities"`
	}
	initParams := map[string]any{
		"pluginId": record.Manifest.ID,
		"dataDir":  record.DataPath,
		"manifest": record.Manifest,
	}
	if err := rpc.call(ctx, "initialize", initParams, 15*time.Second, &initRes); err != nil {
		m.stopProcess(record.Manifest.ID)
		return err
	}
	if initRes.Capabilities != nil {
		record.Runtime = mergeRuntimeCapabilities(&record.Manifest, initRes.Capabilities)
	} else {
		record.Runtime = mergeCapabilities(&record.Manifest)
	}
	if err := validateCapabilities(record.Manifest.Type, record.Runtime); err != nil {
		m.stopProcess(record.Manifest.ID)
		return err
	}
	return nil
}

func (m *Manager) callPluginRPC(ctx context.Context, id, method string, params any, timeout time.Duration, out any) error {
	m.mu.RLock()
	record := m.records[id]
	process := m.processes[id]
	m.mu.RUnlock()
	if record == nil {
		return fmt.Errorf("plugin %s not found", id)
	}
	if process != nil {
		return process.rpc.call(ctx, method, params, timeout, out)
	}

	tempProcess, cleanup, err := m.startTransientProcess(ctx, record)
	if err != nil {
		return err
	}
	defer cleanup()

	return tempProcess.rpc.call(ctx, method, params, timeout, out)
}

func (m *Manager) startTransientProcess(ctx context.Context, record *PluginRecord) (*pluginProcess, func(), error) {
	node, err := m.nodePath()
	if err != nil {
		return nil, nil, err
	}
	mainPath := filepath.Join(record.InstallPath, record.Manifest.Main)
	if _, err := os.Stat(mainPath); err != nil {
		return nil, nil, err
	}

	procCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(procCtx, node, mainPath)
	cmd.Dir = record.InstallPath
	cmd.Env = append(os.Environ(),
		"LEMONTEA_PLUGIN_ID="+record.Manifest.ID,
		"LEMONTEA_PLUGIN_DATA_DIR="+record.DataPath,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, nil, err
	}
	logFile, err := os.OpenFile(filepath.Join(m.logsPath, record.Manifest.ID+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		cancel()
		_ = logFile.Close()
		return nil, nil, err
	}

	rpc := newRPCClient(record.Manifest.ID, stdin, stdout, m.pluginHostRPCHandler(record.Manifest.ID))
	process := &pluginProcess{cmd: cmd, rpc: rpc, cancel: cancel, log: logFile}
	waitDone := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		rpc.close()
		_ = logFile.Close()
		close(waitDone)
	}()

	var initRes struct {
		Capabilities *Capabilities `json:"capabilities"`
	}
	initParams := map[string]any{
		"pluginId": record.Manifest.ID,
		"dataDir":  record.DataPath,
		"manifest": record.Manifest,
	}
	if err := rpc.call(ctx, "initialize", initParams, 15*time.Second, &initRes); err != nil {
		cancel()
		_ = cmd.Process.Kill()
		<-waitDone
		return nil, nil, err
	}

	cleanup := func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		_ = process.rpc.call(shutdownCtx, "shutdown", map[string]any{}, 2*time.Second, nil)
		process.cancel()
		select {
		case <-waitDone:
		case <-shutdownCtx.Done():
			if process.cmd.Process != nil {
				_ = process.cmd.Process.Kill()
			}
			<-waitDone
		}
	}

	return process, cleanup, nil
}

func (m *Manager) stopProcess(id string) {
	m.mu.Lock()
	process := m.processes[id]
	delete(m.processes, id)
	m.mu.Unlock()
	if process == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = process.rpc.call(ctx, "shutdown", map[string]any{}, 2*time.Second, nil)
	process.cancel()
	select {
	case <-process.rpc.closed:
	case <-ctx.Done():
		if process.cmd.Process != nil {
			_ = process.cmd.Process.Kill()
		}
	}
	process.rpc.close()
	_ = process.log.Close()
}

func (m *Manager) pluginHostRPCHandler(pluginID string) rpcHandler {
	return func(ctx context.Context, method string, params json.RawMessage) (any, error) {
		switch method {
		case "get_credential":
			var req struct {
				Scope string `json:"scope"`
				Key   string `json:"key"`
			}
			if err := json.Unmarshal(params, &req); err != nil {
				return nil, err
			}
			value, ok, err := m.credentialStore.Get(m.pluginCredentialKey(pluginID, req.Scope, req.Key))
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"value": value,
				"set":   ok,
			}, nil
		case "set_credential":
			var req struct {
				Scope string `json:"scope"`
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			if err := json.Unmarshal(params, &req); err != nil {
				return nil, err
			}
			if strings.TrimSpace(req.Value) == "" {
				return nil, fmt.Errorf("credential value is required")
			}
			if err := m.credentialStore.Set(m.pluginCredentialKey(pluginID, req.Scope, req.Key), req.Value); err != nil {
				return nil, err
			}
			return map[string]any{"ok": true}, nil
		case "delete_credential":
			var req struct {
				Scope string `json:"scope"`
				Key   string `json:"key"`
			}
			if err := json.Unmarshal(params, &req); err != nil {
				return nil, err
			}
			if err := m.credentialStore.Delete(m.pluginCredentialKey(pluginID, req.Scope, req.Key)); err != nil {
				return nil, err
			}
			return map[string]any{"ok": true}, nil
		default:
			return nil, fmt.Errorf("unknown host rpc method: %s", method)
		}
	}
}

func (m *Manager) pluginCredentialKey(pluginID, scope, key string) string {
	scope = strings.TrimSpace(scope)
	key = strings.TrimSpace(key)
	if scope == "" {
		scope = "default"
	}
	if key == "" {
		key = "secret"
	}
	return pluginID + ":" + scope + ":" + key
}

func (m *Manager) CallTool(ctx context.Context, pluginID, kind, toolID, args string) (string, error) {
	m.mu.RLock()
	process := m.processes[pluginID]
	record := m.records[pluginID]
	m.mu.RUnlock()
	if process == nil || record == nil || !record.Enabled {
		return "", fmt.Errorf("plugin %s is not running", pluginID)
	}
	method := "call_use_tool"
	if kind == "view_tool" {
		method = "call_view_tool"
	}
	var result struct {
		Content any `json:"content"`
	}
	params := map[string]any{
		"toolId": toolID,
		"args":   json.RawMessage(defaultJSON(args)),
	}
	if err := process.rpc.call(ctx, method, params, 2*time.Minute, &result); err != nil {
		return "", err
	}
	raw, err := json.Marshal(result.Content)
	if err != nil {
		return "", err
	}
	if string(raw) == "null" {
		return "", nil
	}
	if s, ok := result.Content.(string); ok {
		return s, nil
	}
	return string(raw), nil
}

func (m *Manager) RunBeforeLLMSend(ctx context.Context, payload BeforeLLMSendPayload) BeforeLLMSendPayload {
	m.forEachHook("before_llm_send", func(process *pluginProcess) {
		var result BeforeLLMSendResult
		if err := process.rpc.call(ctx, "before_llm_send", payload, 10*time.Second, &result); err == nil && len(result.Messages) > 0 {
			payload.Messages = result.Messages
		} else if err != nil {
			logger.Warm("plugin before_llm_send failed:", err)
		}
	})
	return payload
}

func (m *Manager) RunAfterLLMSend(ctx context.Context, payload AfterLLMSendPayload) {
	m.forEachHook("after_llm_send", func(process *pluginProcess) {
		if err := process.rpc.call(ctx, "after_llm_send", payload, 10*time.Second, nil); err != nil {
			logger.Warm("plugin after_llm_send failed:", err)
		}
	})
}

func (m *Manager) forEachHook(hook string, fn func(*pluginProcess)) {
	m.mu.RLock()
	var processes []*pluginProcess
	for id, record := range m.records {
		if !record.Enabled || !hasString(record.Runtime.Hooks, hook) {
			continue
		}
		if process := m.processes[id]; process != nil {
			processes = append(processes, process)
		}
	}
	m.mu.RUnlock()
	for _, process := range processes {
		fn(process)
	}
}

func readManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(path, "plugin.json"))
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	manifest.Raw = raw
	return &manifest, nil
}

func validateManifest(manifest *Manifest) error {
	if manifest == nil || strings.TrimSpace(manifest.ID) == "" || strings.TrimSpace(manifest.Name) == "" || strings.TrimSpace(manifest.Version) == "" || strings.TrimSpace(manifest.Main) == "" {
		return fmt.Errorf("plugin manifest missing required fields")
	}
	if manifest.Type != TypeAgent && manifest.Type != TypeGeneral {
		return fmt.Errorf("unknown plugin type %s", manifest.Type)
	}
	if manifest.Capabilities != nil {
		if err := validateCapabilities(manifest.Type, *manifest.Capabilities); err != nil {
			return err
		}
	}
	return nil
}

func validateCapabilities(pluginType string, caps Capabilities) error {
	if pluginType == TypeAgent && (len(caps.UseTools) > 0 || len(caps.ViewTools) > 0) {
		return fmt.Errorf("agent_plugin cannot register tools")
	}
	if pluginType == TypeGeneral && len(caps.Agents) > 0 {
		return fmt.Errorf("general_plugin cannot register agents")
	}
	return nil
}

func mergeCapabilities(manifest *Manifest) Capabilities {
	caps := Capabilities{}
	if manifest.Capabilities != nil {
		caps = *manifest.Capabilities
	}
	caps.Views = append(caps.Views, manifest.Views...)
	return normalizeCapabilities(caps)
}

func mergeRuntimeCapabilities(manifest *Manifest, runtime *Capabilities) Capabilities {
	caps := Capabilities{}
	if manifest.Capabilities != nil {
		caps = *manifest.Capabilities
	}
	if runtime != nil {
		caps.UseTools = append(caps.UseTools, runtime.UseTools...)
		caps.ViewTools = append(caps.ViewTools, runtime.ViewTools...)
		caps.Agents = append(caps.Agents, runtime.Agents...)
		caps.Views = append(caps.Views, runtime.Views...)
		caps.Hooks = append(caps.Hooks, runtime.Hooks...)
	}
	caps.Views = append(caps.Views, manifest.Views...)
	return normalizeCapabilities(caps)
}

func normalizeCapabilities(caps Capabilities) Capabilities {
	caps.UseTools = uniquePluginTools(caps.UseTools)
	caps.ViewTools = uniquePluginTools(caps.ViewTools)
	caps.Agents = uniquePluginAgents(caps.Agents)
	caps.Views = uniquePluginViews(caps.Views)
	caps.Hooks = uniqueStrings(caps.Hooks)
	return caps
}

func uniquePluginTools(items []PluginTool) []PluginTool {
	if len(items) == 0 {
		return items
	}
	res := make([]PluginTool, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.ID)
		if key == "" {
			key = strings.TrimSpace(item.Name) + "|" + strings.TrimSpace(item.ViewID)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		res = append(res, item)
	}
	return res
}

func uniquePluginAgents(items []PluginAgent) []PluginAgent {
	if len(items) == 0 {
		return items
	}
	res := make([]PluginAgent, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.ID)
		if key == "" {
			key = strings.TrimSpace(item.Name)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		res = append(res, item)
	}
	return res
}

func uniquePluginViews(items []PluginView) []PluginView {
	if len(items) == 0 {
		return items
	}
	res := make([]PluginView, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.ID)
		if key == "" {
			key = strings.TrimSpace(item.Entry)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		res = append(res, item)
	}
	return res
}

func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return items
	}
	res := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		res = append(res, item)
	}
	return res
}

func recordToSummary(record *PluginRecord) Summary {
	return Summary{
		ID:          record.Manifest.ID,
		Name:        record.Manifest.Name,
		Version:     record.Manifest.Version,
		Description: record.Manifest.Description,
		Type:        record.Manifest.Type,
		Author:      record.Manifest.Author,
		Enabled:     record.Enabled,
		Status:      record.Status,
		LastError:   record.LastError,
		HasSettings: record.Manifest.SettingsView != nil,
		Permissions: record.Manifest.Permissions,
		UseTools:    record.Runtime.UseTools,
		ViewTools:   record.Runtime.ViewTools,
		Agents:      record.Runtime.Agents,
		Views:       record.Runtime.Views,
		Hooks:       record.Runtime.Hooks,
	}
}

func copyDir(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

func defaultJSON(s string) string {
	if strings.TrimSpace(s) == "" {
		return "{}"
	}
	return s
}

func hasString(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}

func nodeDownloadURL() (string, error) {
	platform, ext, err := nodePlatform()
	if err != nil {
		return "", err
	}
	filename := fmt.Sprintf("node-%s-%s.%s", pluginNodeVersion, platform, ext)
	return fmt.Sprintf("https://nodejs.org/download/release/%s/%s", pluginNodeVersion, filename), nil
}

func nodePlatform() (string, string, error) {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "arm64"
	default:
		return "", "", fmt.Errorf("unsupported node runtime architecture: %s", runtime.GOARCH)
	}
	switch runtime.GOOS {
	case "darwin":
		return "darwin-" + arch, "tar.gz", nil
	case "linux":
		return "linux-" + arch, "tar.gz", nil
	case "windows":
		return "win-" + arch, "zip", nil
	default:
		return "", "", fmt.Errorf("unsupported node runtime OS: %s", runtime.GOOS)
	}
}

func extractTarGz(archivePath, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target, err := safeExtractPath(dest, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tarReader)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, target); err != nil && !os.IsExist(err) {
				return err
			}
		}
	}
	return nil
}

func extractZip(archivePath, dest string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		target, err := safeExtractPath(dest, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, file.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeInErr := in.Close()
		closeOutErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeInErr != nil {
			return closeInErr
		}
		if closeOutErr != nil {
			return closeOutErr
		}
	}
	return nil
}

func safeExtractPath(dest, name string) (string, error) {
	target := filepath.Join(dest, name)
	cleanDest, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	cleanTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if cleanTarget != cleanDest && !strings.HasPrefix(cleanTarget, cleanDest+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive contains invalid path: %s", name)
	}
	return cleanTarget, nil
}

func firstExtractedDir(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(path, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("node archive did not contain a runtime directory")
}
