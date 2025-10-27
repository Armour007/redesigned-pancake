<script lang="ts">
  import { goto } from '$app/navigation'; // <-- Import goto
  import { API_BASE, authHeaders } from '$lib/api';
  import Alert from '$lib/components/Alert.svelte';

  let email = '';
  let password = '';
  let errorMessage = '';
  let isLoading = false;

  async function handleLogin() {
    isLoading = true;
    errorMessage = '';
    try {
  const response = await fetch(`${API_BASE}/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email, password }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || `HTTP error! status: ${response.status}`);
      }

      // --- START CHANGES ---
      console.log('Login successful, token:', data.token);

      // Store token securely (localStorage is okay for MVP)
      localStorage.setItem('aura_token', data.token);

      // Fetch user's orgs and store first org id for dashboard
      try {
        const orgRes = await fetch(`${API_BASE}/organizations/mine`, {
          headers: authHeaders(data.token)
        });
        if (orgRes.ok) {
          const orgs = await orgRes.json();
          if (Array.isArray(orgs) && orgs.length > 0) {
            localStorage.setItem('aura_org_id', orgs[0].id);
          }
        }
      } catch (_) {}

      // Redirect to the dashboard page
      await goto('/dashboard'); // Use await with goto

      // --- END CHANGES ---

    } catch (error: any) {
      errorMessage = error.message || 'Login failed. Please try again.';
      console.error('Login error:', error);
    } finally {
      isLoading = false;
    }
  }
</script>

<div class="flex items-center justify-center min-h-screen px-4">
  <div class="w-full max-w-md p-8 space-y-6 bg-[#1A1A1A] rounded-xl shadow-lg border border-[#333333]">
    <div class="flex justify-center">
      <div class="w-10 h-10 bg-[#7C3AED] rounded-full"></div>
    </div>
    <h1 class="text-2xl font-bold text-center text-white">
      Welcome Back
    </h1>

    <form on:submit|preventDefault={handleLogin} class="space-y-6">
      <div>
        <label for="email" class="block text-sm font-medium text-gray-300">Email</label>
        <input
          id="email"
          name="email"
          type="email"
          bind:value={email}
          required
          disabled={isLoading}
          class="mt-1 block w-full p-3 bg-[#111111] text-white rounded-lg border border-[#333333] focus:ring-2 focus:ring-[#7C3AED] focus:border-transparent placeholder-gray-500 text-sm"
          placeholder="you@example.com"
        />
      </div>

      <div>
        <label for="password" class="block text-sm font-medium text-gray-300">Password</label>
        <input
          id="password"
          name="password"
          type="password"
          bind:value={password}
          required
          disabled={isLoading}
          class="mt-1 block w-full p-3 bg-[#111111] text-white rounded-lg border border-[#333333] focus:ring-2 focus:ring-[#7C3AED] focus:border-transparent placeholder-gray-500 text-sm"
          placeholder="••••••••"
        />
      </div>

      {#if errorMessage}
        <Alert variant="error">{errorMessage}</Alert>
      {/if}

      <div>
        <button
          type="submit"
          disabled={isLoading}
          class="w-full flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-[#7C3AED] hover:bg-[#6d28d9] focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-[#7C3AED] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {#if isLoading}
            Loading...
          {:else}
            Log In
          {/if}
        </button>
      </div>
    </form>
     <p class="text-sm text-center text-gray-400">
       Don't have an account?
       <a href="/register" class="font-medium text-[#7C3AED] hover:text-[#6d28d9]">
         Sign Up
       </a>
     </p>
  </div>
</div>