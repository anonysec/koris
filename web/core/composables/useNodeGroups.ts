import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface NodeGroup {
  id: number
  name: string
  region: string
  description: string
  load_balancing_enabled: boolean
  max_load_percent: number
  created_at: string
  updated_at: string
}

export interface CreateGroupData {
  name: string
  region?: string
  description?: string
  load_balancing_enabled?: boolean
  max_load_percent?: number
}

export interface LoadOverview {
  groups: Array<{
    id: number
    name: string
    nodes: Array<{
      id: number
      name: string
      load_percent: number
      active_sessions: number
      max_capacity: number
    }>
  }>
}

export interface UseNodeGroupsReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<NodeGroup[]>
  listGroups: () => Promise<NodeGroup[]>
  createGroup: (data: CreateGroupData) => Promise<NodeGroup>
  updateGroup: (id: number, data: Partial<CreateGroupData>) => Promise<NodeGroup>
  deleteGroup: (id: number) => Promise<void>
  assignNode: (nodeId: number, groupId: number) => Promise<void>
  getLoadOverview: () => Promise<LoadOverview>
}

export function useNodeGroups(): UseNodeGroupsReturn {
  const { get, post, patch, del, loading, error } = useApi()
  const data = ref<NodeGroup[]>([]) as Ref<NodeGroup[]>

  async function listGroups(): Promise<NodeGroup[]> {
    const result = await get<{ ok: boolean; groups: NodeGroup[] }>('/api/node-groups')
    data.value = result.groups
    return result.groups
  }

  async function createGroup(groupData: CreateGroupData): Promise<NodeGroup> {
    const result = await post<{ ok: boolean; group: NodeGroup }>('/api/node-groups', groupData)
    return result.group
  }

  async function updateGroup(id: number, groupData: Partial<CreateGroupData>): Promise<NodeGroup> {
    const result = await patch<{ ok: boolean; group: NodeGroup }>(`/api/node-groups/${id}`, groupData)
    return result.group
  }

  async function deleteGroup(id: number): Promise<void> {
    await del(`/api/node-groups/${id}`)
  }

  async function assignNode(nodeId: number, groupId: number): Promise<void> {
    await post(`/api/nodes/${nodeId}/assign-group`, { group_id: groupId })
  }

  async function getLoadOverview(): Promise<LoadOverview> {
    const result = await get<{ ok: boolean } & LoadOverview>('/api/node-groups/load')
    return result
  }

  return {
    loading,
    error,
    data,
    listGroups,
    createGroup,
    updateGroup,
    deleteGroup,
    assignNode,
    getLoadOverview,
  }
}
