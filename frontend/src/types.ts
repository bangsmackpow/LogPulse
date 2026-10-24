export interface DiscoveredLog {
  id: number;
  filePath: string;
  appName: string;
  fileSize: number;
  lastModified: string;
  format: string;
  isFavorite: boolean;
  tag: string;
}

export interface LogLine {
  fileId: string;
  lineNumber: number;
  timestamp: string;
  raw: string;
  level: 'ERROR' | 'WARN' | 'INFO' | 'DEBUG';
}

export interface ScrubMatch {
  type: string;
  original: string;
  replacement: string;
}

export interface AIResponse {
  text: string;
  inputTokens: number;
  outputTokens: number;
  costUsd: number;
}

export interface SessionMarker {
  id: string;
  timestamp: string;
  label: string;
  afterLineNumber: number;
}
