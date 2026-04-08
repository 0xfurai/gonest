import rss from '@astrojs/rss';

export async function GET(context) {
  const modules = import.meta.glob('./docs/*.mdx', { eager: true });

  const items = Object.entries(modules).map(([path, mod]) => {
    const slug = path.replace('./docs/', '').replace(/\.mdx$/, '');
    return {
      title: mod.frontmatter?.title ?? slug,
      description:
        mod.frontmatter?.description ??
        `GoNest documentation: ${mod.frontmatter?.title ?? slug}.`,
      link: `/docs/${slug}/`,
      pubDate: mod.frontmatter?.pubDate
        ? new Date(mod.frontmatter.pubDate)
        : undefined,
    };
  });

  items.sort((a, b) => a.title.localeCompare(b.title));

  return rss({
    title: 'GoNest Documentation',
    description:
      'A progressive Go framework for building efficient, reliable, and scalable server-side applications. Updates to the GoNest documentation.',
    site: context.site,
    items,
    customData: `<language>en-us</language>`,
  });
}
