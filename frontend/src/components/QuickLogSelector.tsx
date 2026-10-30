import React, { useMemo } from 'react';
import { Zap } from 'lucide-react';
import { useLogStore } from '../stores/useLogStore';

interface KnownAppDefinition {
  id: string;
  name: string;
  matcher: (path: string, appName: string) => boolean;
}

const KNOWN_APPS: KnownAppDefinition[] = [
  {
    id: 'discord',
    name: 'Discord',
    matcher: (p, a) => p.toLowerCase().includes('discord') || a.toLowerCase().includes('discord'),
  },
  {
    id: 'chrome',
    name: 'Google Chrome',
    matcher: (p, a) => p.toLowerCase().includes('chrome') || a.toLowerCase().includes('chrome'),
  },
  {
    id: 'vscode',
    name: 'Visual Studio Code',
    matcher: (p, a) =>
      p.toLowerCase().includes('vscode') ||
      p.toLowerCase().includes('.vscode') ||
      p.toLowerCase().includes('code\\logs') ||
      p.toLowerCase().includes('code/logs') ||
      a.toLowerCase() === 'code',
  },
  {
    id: 'slack',
    name: 'Slack',
    matcher: (p, a) => p.toLowerCase().includes('slack') || a.toLowerCase().includes('slack'),
  },
  {
    id: 'docker',
    name: 'Docker Desktop',
    matcher: (p, a) => p.toLowerCase().includes('docker') || a.toLowerCase().includes('docker'),
  },
];

export const QuickLogSelector: React.FC = () => {
  const { logs, switchActiveTails } = useLogStore();

  // Find Windows Event Logs
  const winEventLogs = useMemo(() => {
    const system = logs.find((l) => l.filePath.toLowerCase().endsWith('system.evtx'));
    const app = logs.find((l) => l.filePath.toLowerCase().endsWith('application.evtx'));
    const sec = logs.find((l) => l.filePath.toLowerCase().endsWith('security.evtx'));

    return [
      { name: 'Windows System (System.evtx)', path: system?.filePath || 'C:\\Windows\\System32\\winevt\\Logs\\System.evtx', found: !!system },
      { name: 'Windows Application (Application.evtx)', path: app?.filePath || 'C:\\Windows\\System32\\winevt\\Logs\\Application.evtx', found: !!app },
      { name: 'Windows Security (Security.evtx)', path: sec?.filePath || 'C:\\Windows\\System32\\winevt\\Logs\\Security.evtx', found: !!sec },
    ];
  }, [logs]);

  // Group discovered application logs
  const appGroups = useMemo(() => {
    return KNOWN_APPS.map((app) => {
      const matching = logs.filter((l) => app.matcher(l.filePath, l.appName));
      return {
        ...app,
        paths: matching.map((l) => l.filePath),
        count: matching.length,
      };
    }).filter((g) => g.count > 0);
  }, [logs]);

  const handleSelect = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const val = e.target.value;
    if (!val) return;

    if (val.startsWith('winevt:')) {
      const path = val.replace('winevt:', '');
      const item = winEventLogs.find((w) => w.path === path);
      switchActiveTails([path], item?.name || 'Windows Event Log');
    } else if (val.startsWith('app:')) {
      const appId = val.replace('app:', '');
      const group = appGroups.find((g) => g.id === appId);
      if (group && group.paths.length > 0) {
        switchActiveTails(group.paths, `${group.name} (${group.paths.length} files)`);
      }
    }

    // Reset select to default prompt
    e.target.value = '';
  };

  return (
    <div className="relative flex items-center">
      <div className="relative flex items-center">
        <Zap className="absolute left-2 h-3.5 w-3.5 text-indigo-400 pointer-events-none" />
        <select
          onChange={handleSelect}
          defaultValue=""
          className="h-8 rounded bg-canvas-900 pl-7 pr-8 text-xs text-gray-200 border border-canvas-600 hover:border-canvas-500 focus:border-indigo-500 focus:outline-none cursor-pointer font-sans transition-colors appearance-none"
          title="Quick Jump to Known System & App Logs"
        >
          <option value="" disabled>
            ⚡ Quick Jump to Log...
          </option>

          <optgroup label="── Windows Event Logs ──">
            {winEventLogs.map((item) => (
              <option key={item.path} value={`winevt:${item.path}`}>
                {item.name}
              </option>
            ))}
          </optgroup>

          {appGroups.length > 0 && (
            <optgroup label="── Discovered Applications ──">
              {appGroups.map((group) => (
                <option key={group.id} value={`app:${group.id}`}>
                  {group.name} ({group.count} {group.count === 1 ? 'file' : 'files - multi-tail'})
                </option>
              ))}
            </optgroup>
          )}
        </select>
        <div className="pointer-events-none absolute right-2 text-gray-400 text-[10px]">
          ▼
        </div>
      </div>
    </div>
  );
};
