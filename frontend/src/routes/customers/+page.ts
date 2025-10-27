import { getAllCaseStudies } from '$lib/content/caseStudies';
import { caseStudies as fallback } from '$lib/customers/caseStudies';

export const load = () => {
  const docs = getAllCaseStudies();
  const list = docs.length
    ? docs.map((d) => ({ slug: d.slug, name: d.name, logo: d.logo, headline: d.headline, summary: d.summary }))
    : fallback;
  return { caseStudies: list };
};
