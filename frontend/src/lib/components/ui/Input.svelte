<script lang="ts">
  interface Props {
    label?: string;
    placeholder?: string;
    type?: string;
    value?: string;
    required?: boolean;
    disabled?: boolean;
    error?: string;
    class?: string;
    id?: string;
  }

  let {
    label,
    placeholder = '',
    type = 'text',
    value = $bindable(''),
    required = false,
    disabled = false,
    error,
    class: className = '',
    id
  }: Props = $props();

  const inputId = id || `input-${Math.random().toString(36).substr(2, 9)}`;

  const inputClasses = $derived(`
    block w-full px-3 py-2 border rounded-lg shadow-sm placeholder-gray-500 
    focus:outline-none focus:ring-2 focus:ring-offset-0 transition-colors duration-200
    bg-dark-800 text-gray-200
    ${error 
      ? 'border-red-500 focus:ring-red-500 focus:border-red-500' 
      : 'border-dark-600 focus:ring-primary-500 focus:border-primary-500'
    }
    ${disabled ? 'bg-dark-900 text-gray-600 cursor-not-allowed' : ''}
    ${className}
  `.trim().replace(/\s+/g, ' '));
</script>

<div class="space-y-1">
  {#if label}
    <label for={inputId} class="block text-sm font-medium text-gray-300">
      {label}
      {#if required}
        <span class="text-red-500">*</span>
      {/if}
    </label>
  {/if}
  
  <input
    {id}
    {type}
    {placeholder}
    {required}
    {disabled}
    bind:value
    class={inputClasses}
  />
  
  {#if error}
    <p class="text-sm text-red-400" id="{inputId}-error">
      {error}
    </p>
  {/if}
</div> 