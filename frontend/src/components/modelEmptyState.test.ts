import assert from 'node:assert/strict'
import test from 'node:test'

import { getModelEmptyState } from './modelEmptyState.ts'

test('gives tenant admins an actionable model configuration empty state', () => {
  assert.deepEqual(getModelEmptyState(true), {
    messageKey: 'input.modelEmptyAdmin',
    canConfigure: true,
  })
})

test('directs non-admin users to their tenant administrator', () => {
  assert.deepEqual(getModelEmptyState(false), {
    messageKey: 'input.modelEmptyMember',
    canConfigure: false,
  })
})
