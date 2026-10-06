/** Query keys of the to-do reads (B-N6-08). Every write invalidates `all`. */
export const todoKeys = {
  all: ['todo'] as const,
  board: () => [...todoKeys.all, 'board'] as const,
  task: (id: number) => [...todoKeys.all, 'task', id] as const,
};
