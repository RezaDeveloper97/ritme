// Public API of entities/admin.
export { adminSchema, adminRoleSchema, adminInitial, permissionsSchema, can, ADMIN_ROLES } from './model/admin';
export type { Admin, AdminRole, Permissions, PermissionLevel } from './model/admin';
export { RoleBadge } from './ui/RoleBadge';
