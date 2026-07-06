// The four static info topics this screen can render. The route validates its
// `[topic]` param against this list before mounting the screen.
export const INFO_TOPICS = ['privacy', 'terms', 'about', 'help'] as const;

export type InfoTopic = (typeof INFO_TOPICS)[number];

export function isInfoTopic(value: string): value is InfoTopic {
  return (INFO_TOPICS as readonly string[]).includes(value);
}
