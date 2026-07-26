export interface ModelEmptyState {
  messageKey: 'input.modelEmptyAdmin' | 'input.modelEmptyMember'
  canConfigure: boolean
}

export function getModelEmptyState(canManageModels: boolean): ModelEmptyState {
  return {
    messageKey: canManageModels ? 'input.modelEmptyAdmin' : 'input.modelEmptyMember',
    canConfigure: canManageModels,
  }
}
