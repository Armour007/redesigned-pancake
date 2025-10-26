<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { fade } from 'svelte/transition'; // Import fade transition

  export let showModal: boolean = false;
  export let title: string = 'Modal Title';

  const dispatch = createEventDispatcher();

  function closeModal() {
    dispatch('close');
  }

  // Close modal if backdrop is clicked
  function handleBackdropClick(event: MouseEvent) {
      if (event.target === event.currentTarget) {
          closeModal();
      }
  }

  // Close modal on Escape key press
  function handleKeydown(event: KeyboardEvent) {
      // Only close if the modal is actually showing
      if (showModal && event.key === 'Escape') {
          closeModal();
      }
  }
</script>

<svelte:window on:keydown={handleKeydown}/>

{#if showModal}
  <div
    class="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 transition-opacity duration-300"
    on:click={handleBackdropClick}
    role="dialog"
    aria-modal="true"
    aria-labelledby="modal-title"
    transition:fade={{ duration: 150 }}
  >
    <div
      class="bg-[#1A1A1A] rounded-xl shadow-xl border border-[#333333] w-full max-w-lg relative flex flex-col max-h-[90vh] outline-none"
      role="document"
      tabindex="-1"
    >
      <div class="flex items-center justify-between p-4 border-b border-[#333333] flex-shrink-0">
        <h2 id="modal-title" class="text-lg font-semibold text-white">{title}</h2>
        <button
          on:click={closeModal}
          class="text-gray-400 hover:text-white p-1 rounded-full hover:bg-white/10"
          aria-label="Close modal"
        >
          <span class="material-symbols-outlined text-xl">close</span>
        </button>
      </div>

      <div class="p-6 flex-grow overflow-y-auto">
        <slot /> </div>
    </div>
  </div>
{/if}