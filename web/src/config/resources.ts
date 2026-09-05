import type { Field, ResourceKind } from '../types/api';

export const resourcePaths = {
  database: '/api/v1/databases',
  linux: '/api/v1/linux/hosts',
  kubernetes: '/api/v1/kubernetes/clusters',
};
const environmentField: Field = { name: 'environment' };
export const resourceFields: Record<ResourceKind, Field[]> = {
  database: [
    { name: 'name' },
    environmentField,
    { name: 'driver', options: ['postgres', 'mysql'] },
  ],
  linux: [{ name: 'name' }, environmentField, { name: 'address' }, { name: 'user' }],
  kubernetes: [{ name: 'name' }, environmentField, { name: 'context' }],
};
