<script lang="ts">
  import { API_BASE } from '$lib/api';
  import { redirect } from '@sveltejs/kit';
  // SSR redirect to backend Swagger UI
  export const load = () => {
    throw redirect(302, `${API_BASE}/docs`);
  };
</script>
