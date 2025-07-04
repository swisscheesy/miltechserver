<script lang="ts">
  import Input from '$lib/components/ui/Input.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Alert from '$lib/components/ui/Alert.svelte';
  import { deleteAccountWithCredentials } from '$lib/services/auth';

  let email = '';
  let password = '';
  let isLoading = false;
  let alertMessage = '';
  let alertVariant: 'success' | 'error' | 'warning' | 'info' = 'info';
  let showAlert = false;
  let accountDeleted = false;

  async function handleSubmit() {
    isLoading = true;
    showAlert = false;
    
    try {
      const result = await deleteAccountWithCredentials(email, password);
      
      if (result.success) {
        alertVariant = 'success';
        alertMessage = result.message;
        accountDeleted = true;
        // Clear form data for security
        email = '';
        password = '';
      } else {
        alertVariant = 'error';
        alertMessage = result.message;
      }
    } catch (error) {
      alertVariant = 'error';
      alertMessage = 'An unexpected error occurred. Please try again.';
      console.error('Account deletion error:', error);
    } finally {
      isLoading = false;
      showAlert = true;
    }
  }

  function dismissAlert() {
    showAlert = false;
  }
</script>

<svelte:head>
  <title>Account Deletion - Miltech</title>
  <meta name="description" content="Delete your Miltech account permanently." />
</svelte:head>

<!-- Main Content -->
<section class="min-h-screen bg-gradient-to-br from-dark-900 via-dark-800 to-primary-900 flex items-center justify-center px-4 sm:px-6 lg:px-8">
  <div class="w-full max-w-md">
    <div class="bg-dark-800 border border-dark-700 rounded-xl shadow-2xl p-8">
      <!-- Header -->
      <div class="text-center mb-8">
        <h1 class="text-3xl font-bold text-white mb-2">
          Delete Account
        </h1>
        <p class="text-gray-400 text-sm">
          This action cannot be undone. Please confirm your identity to proceed.
        </p>
      </div>

      <!-- Alert Message -->
      {#if showAlert}
        <div class="mb-6">
          <Alert variant={alertVariant} dismissible onDismiss={dismissAlert}>
            {alertMessage}
          </Alert>
        </div>
      {/if}

      {#if !accountDeleted}
        <!-- Form -->
        <form on:submit|preventDefault={handleSubmit} class="space-y-6">
          <!-- Email Field -->
          <Input
            label="Email Address"
            type="email"
            placeholder="Enter your email address"
            bind:value={email}
            required
            disabled={isLoading}
          />

          <!-- Password Field -->
          <Input
            label="Password"
            type="password"
            placeholder="Enter your password"
            bind:value={password}
            required
            disabled={isLoading}
          />

          <!-- Warning Message -->
          <div class="bg-red-900/20 border border-red-500/30 rounded-lg p-4">
            <div class="flex items-start">
              <svg class="h-5 w-5 text-red-400 mt-0.5 mr-3 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
              </svg>
              <div>
                <h3 class="text-sm font-medium text-red-300 mb-1">
                  Warning: Account Deletion
                </h3>
                <p class="text-sm text-red-400">
                  This will permanently delete your account and all associated data. This action cannot be reversed.
                </p>
              </div>
            </div>
          </div>

          <!-- Submit Button -->
          <Button
            type="submit"
            variant="danger"
            size="lg"
            class="w-full"
            loading={isLoading}
            disabled={!email || !password || isLoading}
          >
            {isLoading ? 'Processing...' : 'Delete My Account'}
          </Button>

          <!-- Cancel Link -->
          <div class="text-center">
            <a
              href="/"
              class="text-sm text-gray-400 hover:text-gray-300 transition-colors duration-200"
            >
              Cancel and return to homepage
            </a>
          </div>
        </form>
      {:else}
        <!-- Success State -->
        <div class="text-center space-y-4">
          <div class="mx-auto w-16 h-16 bg-green-500/20 rounded-full flex items-center justify-center">
            <svg class="w-8 h-8 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
            </svg>
          </div>
          <div>
            <h3 class="text-lg font-medium text-white mb-2">Account Deleted Successfully</h3>
            <p class="text-gray-400 text-sm mb-6">
              Your account and all associated data have been permanently removed.
            </p>
            <Button
              onclick={() => window.location.href = '/'}
              variant="primary"
              size="md"
            >
              Return to Homepage
            </Button>
          </div>
        </div>
      {/if}
    </div>
  </div>
</section>