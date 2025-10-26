<script lang="ts">
  import '../app.css'; // Imports Tailwind styles
</script>

<div class="min-h-screen bg-[#111111] text-white font-sans">
  <slot /> </div>

<style lang="postcss">
  :global(body) {
    font-family: 'Inter', sans-serif; /* Optional: Ensure Inter font is used if Tailwind doesn't pick it up */
  }
</style>
