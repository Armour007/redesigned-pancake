// Ambient module declarations to satisfy svelte-check for $env dynamic modules
// SvelteKit normally provides these in .svelte-kit/types, but this helps when svelte-check
// can't resolve them for some reason in editor/CI.
declare module '$env/dynamic/public' {
  export const env: Record<string, string | undefined>;
}

declare module '$env/dynamic/private' {
  export const env: Record<string, string | undefined>;
}
