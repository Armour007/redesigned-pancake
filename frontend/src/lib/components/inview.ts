export type InViewOptions = {
  threshold?: number;
  rootMargin?: string;
  once?: boolean;
  className?: string;
};

export function inview(node: HTMLElement, opts: InViewOptions = {}) {
  const { threshold = 0.15, rootMargin = '0px', once = true, className = 'inview' } = opts;

  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          node.classList.add(className);
          if (once) observer.unobserve(node);
        }
      });
    },
    { threshold, rootMargin }
  );

  observer.observe(node);

  return {
    destroy() {
      observer.disconnect();
    }
  };
}
