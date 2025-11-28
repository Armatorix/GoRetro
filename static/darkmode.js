// darkmode.js - Dark mode support for GoRetro
(function() {
    const STORAGE_KEY = 'goretro_theme';
    const THEME_DARK = 'dark';
    const THEME_LIGHT = 'light';
    
    let currentTheme = THEME_LIGHT;
    
    /**
     * Get system preference for dark mode
     */
    function getSystemPreference() {
        if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
            return THEME_DARK;
        }
        return THEME_LIGHT;
    }
    
    /**
     * Get stored theme preference
     */
    function getStoredTheme() {
        // Try localStorage first
        if (typeof localStorage !== 'undefined') {
            const stored = localStorage.getItem(STORAGE_KEY);
            if (stored && (stored === THEME_DARK || stored === THEME_LIGHT)) {
                return stored;
            }
        }
        
        // Try cookie
        const cookies = document.cookie.split(';');
        for (let cookie of cookies) {
            const [name, value] = cookie.trim().split('=');
            if (name === STORAGE_KEY && (value === THEME_DARK || value === THEME_LIGHT)) {
                return value;
            }
        }
        
        return null;
    }
    
    /**
     * Save theme preference
     */
    function saveTheme(theme) {
        // Save to localStorage
        if (typeof localStorage !== 'undefined') {
            localStorage.setItem(STORAGE_KEY, theme);
        }
        
        // Save to cookie (expires in 1 year)
        const expires = new Date();
        expires.setFullYear(expires.getFullYear() + 1);
        document.cookie = `${STORAGE_KEY}=${theme}; expires=${expires.toUTCString()}; path=/`;
    }
    
    /**
     * Apply theme to document
     */
    function applyTheme(theme) {
        if (theme === THEME_DARK) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
        
        // Update theme-color meta tag if exists
        const metaThemeColor = document.querySelector('meta[name="theme-color"]');
        if (metaThemeColor) {
            metaThemeColor.setAttribute('content', theme === THEME_DARK ? '#1f2937' : '#ffffff');
        }
        
        currentTheme = theme;
        
        // Update toggle button if it exists
        updateToggleButton();
        
        // Trigger custom event for theme changes
        window.dispatchEvent(new CustomEvent('themeChanged', { detail: { theme: theme } }));
    }
    
    /**
     * Initialize theme
     */
    function initTheme() {
        // Priority: stored > system > default (light)
        const stored = getStoredTheme();
        const theme = stored || getSystemPreference();
        
        applyTheme(theme);
        saveTheme(theme);
        
        return theme;
    }
    
    /**
     * Toggle between light and dark themes
     */
    function toggleTheme() {
        const newTheme = currentTheme === THEME_DARK ? THEME_LIGHT : THEME_DARK;
        applyTheme(newTheme);
        saveTheme(newTheme);
    }
    
    /**
     * Set specific theme
     */
    function setTheme(theme) {
        if (theme !== THEME_DARK && theme !== THEME_LIGHT) {
            console.error(`Invalid theme: ${theme}`);
            return;
        }
        
        applyTheme(theme);
        saveTheme(theme);
    }
    
    /**
     * Get current theme
     */
    function getCurrentTheme() {
        return currentTheme;
    }
    
    /**
     * Check if dark mode is enabled
     */
    function isDarkMode() {
        return currentTheme === THEME_DARK;
    }
    
    /**
     * Update toggle button state
     */
    function updateToggleButton() {
        const toggleButton = document.getElementById('theme-toggle');
        if (!toggleButton) return;
        
        const sunIcon = toggleButton.querySelector('.sun-icon');
        const moonIcon = toggleButton.querySelector('.moon-icon');
        
        if (currentTheme === THEME_DARK) {
            if (sunIcon) sunIcon.classList.remove('hidden');
            if (moonIcon) moonIcon.classList.add('hidden');
            toggleButton.setAttribute('aria-label', 'Switch to light mode');
        } else {
            if (sunIcon) sunIcon.classList.add('hidden');
            if (moonIcon) moonIcon.classList.remove('hidden');
            toggleButton.setAttribute('aria-label', 'Switch to dark mode');
        }
    }
    
    /**
     * Initialize theme toggle button
     */
    function initThemeToggle() {
        const toggleButton = document.getElementById('theme-toggle');
        if (toggleButton) {
            toggleButton.addEventListener('click', toggleTheme);
            updateToggleButton();
        }
    }
    
    /**
     * Listen for system theme changes
     */
    function initSystemThemeListener() {
        if (window.matchMedia) {
            const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
            
            // Modern API
            if (mediaQuery.addEventListener) {
                mediaQuery.addEventListener('change', (e) => {
                    // Only update if user hasn't set a preference
                    const stored = getStoredTheme();
                    if (!stored) {
                        applyTheme(e.matches ? THEME_DARK : THEME_LIGHT);
                    }
                });
            } 
            // Older API
            else if (mediaQuery.addListener) {
                mediaQuery.addListener((e) => {
                    const stored = getStoredTheme();
                    if (!stored) {
                        applyTheme(e.matches ? THEME_DARK : THEME_LIGHT);
                    }
                });
            }
        }
    }
    
    // Export to window
    window.darkMode = {
        toggle: toggleTheme,
        setTheme: setTheme,
        getCurrentTheme: getCurrentTheme,
        isDarkMode: isDarkMode,
        initTheme: initTheme,
        initThemeToggle: initThemeToggle,
        THEME_DARK: THEME_DARK,
        THEME_LIGHT: THEME_LIGHT
    };
    
    // Auto-initialize theme as early as possible to prevent flash
    initTheme();
    
    // Initialize toggle and system listener when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => {
            initThemeToggle();
            initSystemThemeListener();
        });
    } else {
        initThemeToggle();
        initSystemThemeListener();
    }
})();
