import { caseStudies } from '$lib/customers/caseStudies';
import { getAllCaseStudies } from '$lib/content/caseStudies';

export const load = ({ params }) => {
  const docs = getAllCaseStudies();
  const md = docs.find((d) => d.slug === params.slug);
  if (md) {
    return { cs: { ...md, bodyHtml: '', component: md.component } };
  }
  const cs = caseStudies.find((c) => c.slug === params.slug) ?? null;
  return { cs };
};
