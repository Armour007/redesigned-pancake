export type CaseStudyFrontmatter = {
  slug: string;
  name: string;
  logo: string;
  headline: string;
  summary: string;
  metrics?: { label: string; value: string }[];
};

export type CaseStudyDoc = CaseStudyFrontmatter & {
  component: any;
};

export function getAllCaseStudies(): CaseStudyDoc[] {
  const modules = import.meta.glob('/src/content/case-studies/*.{md,svx}', { eager: true }) as Record<string, any>;
  const docs: CaseStudyDoc[] = Object.values(modules).map((m: any) => ({
    ...(m.metadata || {}),
    component: m.default
  }));
  // keep stable order by name, or sort by slug
  docs.sort((a, b) => a.name.localeCompare(b.name));
  return docs;
}
