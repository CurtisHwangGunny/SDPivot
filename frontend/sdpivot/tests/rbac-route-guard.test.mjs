import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const routerSource = await readFile(new URL('../src/router/index.ts', import.meta.url), 'utf8')
const jwtSource = await readFile(new URL('../src/utils/jwt.ts', import.meta.url), 'utf8')

test('admin route roles match the RBAC v2.0 contract', () => {
  assert.match(routerSource, /const allAccessRoles = \['super_admin', 'department_admin', 'knowledge_editor', 'knowledge_viewer'\]/)

  const expectedRoutes = [
    ["'/admin'", "['super_admin', 'department_admin']"],
    ["'/admin/people'", "['super_admin', 'department_admin']"],
    ["'/admin/tags'", 'allAccessRoles'],
    ["'/admin/models'", "['super_admin']"],
    ["'/admin/security'", "['super_admin']"],
    ["'/admin/departments'", "['super_admin', 'department_admin']"],
    ["'/admin/audit'", 'allAccessRoles'],
    ["'/admin/usage'", "['super_admin', 'department_admin']"],
  ]

  for (const [path, roles] of expectedRoutes) {
    assert.equal(routerSource.includes(`  ${path}: ${roles},`), true, `missing RBAC mapping for ${path}`)
  }
})

test('OP guard redirects unauthorized and unknown admin routes to spaces', () => {
  assert.match(routerSource, /if \(to\.path === '\/admin' \|\| to\.path\.startsWith\('\/admin\/'\)\)/)
  assert.match(routerSource, /const allowedRoles = adminRouteRoles\[to\.path\]/)
  assert.match(routerSource, /if \(!allowedRoles\?\.includes\(getRoleFromToken\(\)\)\) return \{ name: 'spaces' \}/)
})

test('JWT role parsing defaults invalid or role-less tokens to knowledge_viewer', () => {
  assert.match(jwtSource, /localStorage\.getItem\(STORAGE_KEYS\.accessToken\)/)
  assert.match(jwtSource, /JSON\.parse\(atob\(token\.split\('\.'\)\[1\]\)\)/)
  assert.match(jwtSource, /return payload\.role \|\| 'knowledge_viewer'/)
  assert.match(jwtSource, /catch \{\s*return 'knowledge_viewer'\s*\}/)
})
