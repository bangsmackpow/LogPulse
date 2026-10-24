export interface SearchMessage {
  lines: Array<{
    lineNumber: number;
    level: string;
    raw: string;
    timestamp: string;
  }>;
  searchTerm: string;
  filterLevel: string[];
}

export interface SearchResult {
  filteredIndices: number[];
}

self.onmessage = (e: MessageEvent<SearchMessage>) => {
  const { lines, searchTerm, filterLevel } = e.data;
  const levelSet = new Set(filterLevel);
  const filteredIndices: number[] = [];

  let regex: RegExp | null = null;
  if (searchTerm.trim() !== '') {
    try {
      regex = new RegExp(searchTerm, 'i');
    } catch {
      // If invalid regex, fallback to substring matching
      regex = null;
    }
  }

  const termLower = searchTerm.toLowerCase();

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (!levelSet.has(line.level)) {
      continue;
    }

    if (searchTerm.trim() === '') {
      filteredIndices.push(i);
      continue;
    }

    if (regex) {
      if (regex.test(line.raw)) {
        filteredIndices.push(i);
      }
    } else {
      if (line.raw.toLowerCase().includes(termLower)) {
        filteredIndices.push(i);
      }
    }
  }

  self.postMessage({ filteredIndices });
};
