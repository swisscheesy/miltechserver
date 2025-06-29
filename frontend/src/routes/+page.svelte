<script lang="ts">
  // No imports needed for simple app store links page
</script>

<svelte:head>
  <title>MilTech - Mobile App</title>
  <meta name="description" content="Download MilTech mobile app for iOS and Android devices." />
</svelte:head>

<!-- Main Content -->
<section class="min-h-screen bg-gradient-to-br from-dark-900 via-dark-800 to-primary-900 flex items-center justify-center">
  <div class="text-center px-4 sm:px-6 lg:px-8">
    <h1 class="text-4xl md:text-5xl font-bold text-white mb-4">
      MilTech
    </h1>
    
    <p class="text-lg text-gray-400 mb-12 max-w-md mx-auto">
      Download our mobile app today.
    </p>
    
    <div class="flex flex-col sm:flex-row gap-6 justify-center items-center">
      <!-- Google Play Store -->
      <a
        href="https://play.google.com/store/apps/details?id=com.fromage.miltech"
        target="_blank"
        rel="noopener noreferrer"
        class="group flex items-center space-x-3 bg-dark-800 hover:bg-dark-700 border border-dark-700 hover:border-primary-500/50 rounded-lg px-6 py-4 transition-all duration-200 hover:shadow-xl hover:shadow-primary-500/20"
      >
        <!-- Google Play Icon -->
        <svg class="h-8 w-8" viewBox="0 0 24 24" fill="none">
          <path d="M3 20.5V3.5C3 2.91 3.34 2.39 3.84 2.15L13.69 12L3.84 21.85C3.34 21.61 3 21.09 3 20.5Z" fill="#EA4335"/>
          <path d="M16.81 15.12L6.05 21.34C5.77 21.5 5.42 21.5 5.12 21.34L3.84 21.85L13.69 12L16.81 15.12Z" fill="#FBBC04"/>
          <path d="M20.16 10.85C20.5 11.05 20.5 11.95 20.16 12.15L16.81 8.88L13.69 12L16.81 15.12L20.16 10.85Z" fill="#4285F4"/>
          <path d="M6.05 2.66C5.77 2.5 5.42 2.5 5.12 2.66L3.84 2.15L13.69 12L16.81 8.88L6.05 2.66Z" fill="#34A853"/>
        </svg>
        
        <div class="text-left">
          <div class="text-xs text-gray-400 group-hover:text-gray-300 transition-colors">
            GET IT ON
          </div>
          <div class="text-lg font-semibold text-white">
            Google Play
          </div>
        </div>
      </a>
      
      <!-- Apple App Store -->
      <a
        href="https://apps.apple.com/us/app/miltech/id1530711439"
        target="_blank"
        rel="noopener noreferrer"
        class="group flex items-center space-x-3 bg-dark-800 hover:bg-dark-700 border border-dark-700 hover:border-primary-500/50 rounded-lg px-6 py-4 transition-all duration-200 hover:shadow-xl hover:shadow-primary-500/20"
      >
        <!-- Apple App Store Icon -->
        <svg class="h-8 w-8" viewBox="0 0 24 24" fill="none">
          <path d="M18.71 19.5C17.88 20.74 17 21.95 15.66 21.97C14.32 22 13.89 21.18 12.37 21.18C10.84 21.18 10.37 21.95 9.09 22C7.85 22.05 6.8 20.68 5.96 19.47C4.25 17 2.94 12.45 4.7 9.39C5.57 7.87 7.13 6.91 8.82 6.88C10.1 6.86 11.32 7.75 12.11 7.75C12.89 7.75 14.37 6.68 15.92 6.84C16.57 6.87 18.39 7.1 19.56 8.82C19.47 8.88 17.39 10.19 17.41 12.63C17.44 15.65 20.06 16.66 20.09 16.67C20.06 16.74 19.67 18.11 18.71 19.5ZM13 3.5C13.73 2.67 14.94 2.04 15.94 2C16.07 3.17 15.6 4.35 14.9 5.19C14.21 6.04 13.07 6.7 11.95 6.61C11.8 5.46 12.36 4.26 13 3.5Z" fill="currentColor" class="text-white"/>
        </svg>
        
        <div class="text-left">
          <div class="text-xs text-gray-400 group-hover:text-gray-300 transition-colors">
            Download on the
          </div>
          <div class="text-lg font-semibold text-white">
            App Store
          </div>
        </div>
      </a>
    </div>
  </div>
</section>
