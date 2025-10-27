<script lang="ts">
  import { goto } from '$app/navigation'; // Import goto for redirection
  import { API_BASE } from '$lib/api';
  import Alert from '$lib/components/Alert.svelte';

  let fullName = '';
  let email = '';
  let password = '';
  let errorMessage = '';
  let successMessage = '';
  let isLoading = false;

  async function handleRegister() {
    isLoading = true;
    errorMessage = '';
    successMessage = '';
    try {
      const response = await fetch(`${API_BASE}/auth/register`, { // Backend register URL
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ full_name: fullName, email, password }), // Use full_name here
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || `HTTP error! status: ${response.status}`);
      }

      // SUCCESS! Show message and maybe redirect to login after a delay
      console.log('Registration successful:', data);
      // Persist organization id for subsequent API calls
      if (data && data.organization_id) {
        try { localStorage.setItem('aura_org_id', data.organization_id); } catch {}
      }
      successMessage = 'Registration successful! Redirecting to login...';
      // Redirect to login after a short delay
      setTimeout(() => {
        goto('/login');
      }, 2000); // Redirect after 2 seconds

    } catch (error: any) {
      errorMessage = error.message || 'Registration failed. Please try again.';
      console.error('Registration error:', error);
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
      Create Your AURA Account
    </h1>

    {#if successMessage}
      <Alert variant="success" className="text-center">{successMessage}</Alert>
    {/if}

    <form on:submit|preventDefault={handleRegister} class="space-y-6">
       <div>
         <label for="fullName" class="block text-sm font-medium text-gray-300">Full Name</label>
         <input
           id="fullName"
           name="fullName"
           type="text"
           bind:value={fullName}
           required
           disabled={isLoading}
           class="mt-1 block w-full p-3 bg-[#111111] text-white rounded-lg border border-[#333333] focus:ring-2 focus:ring-[#7C3AED] focus:border-transparent placeholder-gray-500 text-sm"
           placeholder="Your Name"
         />
       </div>

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
          minlength="8"
          bind:value={password}
          required
          disabled={isLoading}
          class="mt-1 block w-full p-3 bg-[#111111] text-white rounded-lg border border-[#333333] focus:ring-2 focus:ring-[#7C3AED] focus:border-transparent placeholder-gray-500 text-sm"
          placeholder="•••••••• (min 8 characters)"
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
            Creating Account...
          {:else}
            Sign Up
          {/if}
        </button>
      </div>
    </form>
     <p class="text-sm text-center text-gray-400">
       Already have an account?
       <a href="/login" class="font-medium text-[#7C3AED] hover:text-[#6d28d9]">
         Log In
       </a>
     </p>
  </div>
</div>