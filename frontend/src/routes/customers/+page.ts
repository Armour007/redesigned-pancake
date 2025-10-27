import { getAllCaseStudies } from '$lib/content/caseStudies';

export const load = () => {
  const docs = getAllCaseStudies();
  const list = docs.map((d) => ({ slug: d.slug, name: d.name, logo: d.logo, headline: d.headline, summary: d.summary }));
  return { caseStudies: list };
};
