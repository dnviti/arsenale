import type { ConnectionData } from '@/api/connections.api';
import type { Folder } from '@/store/connectionsStore';

export const BASE_PADDING_PX = 16;
export const INDENT_PADDING_PX = 16;

export function depthPl(depth: number) {
  return BASE_PADDING_PX + depth * INDENT_PADDING_PX;
}

export interface FolderNode {
  folder: Folder;
  children: FolderNode[];
}

export function matchesSearch(conn: ConnectionData, query: string): boolean {
  const normalizedQuery = query.toLowerCase();
  return conn.name.toLowerCase().includes(normalizedQuery)
    || conn.host.toLowerCase().includes(normalizedQuery)
    || conn.type.toLowerCase().includes(normalizedQuery)
    || (conn.description?.toLowerCase().includes(normalizedQuery) ?? false);
}

export function pruneFolderTree(
  nodes: FolderNode[],
  folderMap: Map<string, ConnectionData[]>,
): FolderNode[] {
  return nodes.reduce<FolderNode[]>((accumulator, node) => {
    const prunedChildren = pruneFolderTree(node.children, folderMap);
    const hasConnections = (folderMap.get(node.folder.id) || []).length > 0;
    if (hasConnections || prunedChildren.length > 0) {
      accumulator.push({ ...node, children: prunedChildren });
    }
    return accumulator;
  }, []);
}

export function buildFolderTree(folders: Folder[]): FolderNode[] {
  const nodeMap = new Map<string, FolderNode>();
  for (const folder of folders) {
    nodeMap.set(folder.id, { folder, children: [] });
  }

  const roots: FolderNode[] = [];
  for (const node of nodeMap.values()) {
    if (node.folder.parentId && nodeMap.has(node.folder.parentId)) {
      nodeMap.get(node.folder.parentId)?.children.push(node);
    } else {
      roots.push(node);
    }
  }

  return roots;
}

export function collectFolderConnections(
  folderId: string,
  folderMap: Map<string, ConnectionData[]>,
  folders: Folder[],
  recursive: boolean,
): ConnectionData[] {
  const directConnections = folderMap.get(folderId) || [];
  if (!recursive) {
    return [...directConnections];
  }

  const result = [...directConnections];
  const childFolders = folders.filter((folder) => folder.parentId === folderId);
  for (const childFolder of childFolders) {
    result.push(...collectFolderConnections(childFolder.id, folderMap, folders, true));
  }
  return result;
}

export function folderHasSubfolders(folderId: string, folders: Folder[]): boolean {
  return folders.some((folder) => folder.parentId === folderId);
}
