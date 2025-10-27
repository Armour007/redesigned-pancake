import type { PageLoad } from './$types';
import { caseStudies } from '$lib/customers/caseStudies';

export const load: PageLoad = ({ params }) => {
  const cs = caseStudies.find((c) => c.slug === params.slug) ?? null;
  return { cs };
};
