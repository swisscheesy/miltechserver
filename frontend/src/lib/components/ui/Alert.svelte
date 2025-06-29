<script lang="ts">
  import { AlertCircle, CheckCircle, Info, XCircle, X } from 'lucide-svelte';

  interface Props {
    variant?: 'success' | 'error' | 'warning' | 'info';
    dismissible?: boolean;
    class?: string;
    onDismiss?: () => void;
    children?: import('svelte').Snippet;
  }

  let {
    variant = 'info',
    dismissible = false,
    class: className = '',
    onDismiss,
    children
  }: Props = $props();

  const variantClasses = {
    success: 'bg-accent-500/10 text-accent-400 border-accent-500/20',
    error: 'bg-red-500/10 text-red-400 border-red-500/20',
    warning: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20',
    info: 'bg-primary-500/10 text-primary-400 border-primary-500/20'
  };

  const iconComponents = {
    success: CheckCircle,
    error: XCircle,
    warning: AlertCircle,
    info: Info
  };

  const IconComponent = iconComponents[variant];

  const alertClasses = $derived(`p-4 rounded-lg text-sm border ${variantClasses[variant]} ${className}`);
</script>

<div class={alertClasses} role="alert">
  <div class="flex">
    <div class="flex-shrink-0">
      <IconComponent class="h-5 w-5" />
    </div>
    <div class="ml-3 flex-1">
      {#if children}
        {@render children()}
      {/if}
    </div>
    {#if dismissible && onDismiss}
      <div class="ml-auto pl-3">
        <div class="-mx-1.5 -my-1.5">
          <button
            type="button"
            class="inline-flex rounded-md p-1.5 hover:bg-black hover:bg-opacity-10 focus:outline-none focus:ring-2 focus:ring-offset-2"
            onclick={onDismiss}
          >
            <X class="h-4 w-4" />
            <span class="sr-only">Dismiss</span>
          </button>
        </div>
      </div>
    {/if}
  </div>
</div> 