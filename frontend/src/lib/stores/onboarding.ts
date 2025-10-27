import { writable } from 'svelte/store';

const KEY = 'aura_onboarding_completed';

function createOnboardingStore() {
  const { subscribe, set } = writable<boolean>(false);

  const init = () => {
    if (typeof localStorage === 'undefined') return; // SSR guard
    const v = localStorage.getItem(KEY);
    set(v === '1');
  };

  const complete = () => {
    if (typeof localStorage === 'undefined') return;
    localStorage.setItem(KEY, '1');
    set(true);
  };

  const reset = () => {
    if (typeof localStorage === 'undefined') return;
    localStorage.removeItem(KEY);
    set(false);
  };

  return { subscribe, init, complete, reset };
}

export const onboarding = createOnboardingStore();
