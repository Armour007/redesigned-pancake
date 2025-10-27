import { getAllCaseStudies } from '$lib/content/caseStudies';

export const load = ({ params }) => {
  const docs = getAllCaseStudies();
  const md = docs.find((d) => d.slug === params.slug);
  const cs = md ? { ...md, bodyHtml: '', component: md.component } : null;
  return { cs };
};
