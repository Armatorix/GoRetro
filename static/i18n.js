// i18n.js - Internationalization support for GoRetro
(function() {
    const STORAGE_KEY = 'goretro_language';
    const DEFAULT_LANGUAGE = 'en';
    const SUPPORTED_LANGUAGES = ['en', 'pl'];
    
    // Translation storage
    window.translations = window.translations || {};
    
    // Current language
    let currentLanguage = DEFAULT_LANGUAGE;
    
    /**
     * Get browser's preferred language
     */
    function getBrowserLanguage() {
        const browserLang = navigator.language || navigator.userLanguage;
        const shortLang = browserLang.split('-')[0].toLowerCase();
        return SUPPORTED_LANGUAGES.includes(shortLang) ? shortLang : DEFAULT_LANGUAGE;
    }
    
    /**
     * Get language from localStorage or cookie
     */
    function getStoredLanguage() {
        // Try localStorage first
        if (typeof localStorage !== 'undefined') {
            const stored = localStorage.getItem(STORAGE_KEY);
            if (stored && SUPPORTED_LANGUAGES.includes(stored)) {
                return stored;
            }
        }
        
        // Try cookie
        const cookies = document.cookie.split(';');
        for (let cookie of cookies) {
            const [name, value] = cookie.trim().split('=');
            if (name === STORAGE_KEY && SUPPORTED_LANGUAGES.includes(value)) {
                return value;
            }
        }
        
        return null;
    }
    
    /**
     * Save language preference
     */
    function saveLanguage(lang) {
        // Save to localStorage
        if (typeof localStorage !== 'undefined') {
            localStorage.setItem(STORAGE_KEY, lang);
        }
        
        // Save to cookie (expires in 1 year)
        const expires = new Date();
        expires.setFullYear(expires.getFullYear() + 1);
        document.cookie = `${STORAGE_KEY}=${lang}; expires=${expires.toUTCString()}; path=/`;
    }
    
    /**
     * Initialize language
     */
    function initLanguage() {
        // Priority: stored > browser > default
        const stored = getStoredLanguage();
        if (stored) {
            currentLanguage = stored;
        } else {
            currentLanguage = getBrowserLanguage();
            saveLanguage(currentLanguage);
        }
        
        // Update HTML lang attribute
        document.documentElement.lang = currentLanguage;
        
        return currentLanguage;
    }
    
    /**
     * Translate a key to current language
     * Supports nested keys with dot notation: "nav.title"
     * Supports placeholders: t('welcome', { name: 'John' })
     */
    function t(key, params = {}) {
        const keys = key.split('.');
        let translation = window.translations[currentLanguage];
        
        if (!translation) {
            console.warn(`Translations not loaded for language: ${currentLanguage}`);
            return key;
        }
        
        // Navigate through nested object
        for (let k of keys) {
            if (translation && typeof translation === 'object' && k in translation) {
                translation = translation[k];
            } else {
                console.warn(`Translation key not found: ${key} for language: ${currentLanguage}`);
                return key;
            }
        }
        
        // Replace placeholders
        if (typeof translation === 'string') {
            return translation.replace(/\{(\w+)\}/g, (match, param) => {
                return params[param] !== undefined ? params[param] : match;
            });
        }
        
        return translation;
    }
    
    /**
     * Set current language and re-translate page
     */
    function setLanguage(lang) {
        if (!SUPPORTED_LANGUAGES.includes(lang)) {
            console.error(`Unsupported language: ${lang}`);
            return;
        }
        
        currentLanguage = lang;
        saveLanguage(lang);
        
        // Update HTML lang attribute
        document.documentElement.lang = lang;
        
        // Update language selector if it exists
        const selector = document.getElementById('language-selector');
        if (selector && selector.value !== lang) {
            selector.value = lang;
        }
        
        // Trigger custom event for page-specific translations
        window.dispatchEvent(new CustomEvent('languageChanged', { detail: { language: lang } }));
        
        // Re-translate all elements with data-i18n attribute
        translatePage();
    }
    
    /**
     * Get current language
     */
    function getCurrentLanguage() {
        return currentLanguage;
    }
    
    /**
     * Translate all elements with data-i18n attribute
     */
    function translatePage() {
        document.querySelectorAll('[data-i18n]').forEach(element => {
            const key = element.getAttribute('data-i18n');
            const translation = t(key);
            
            if (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA') {
                element.placeholder = translation;
            } else {
                element.textContent = translation;
            }
        });
        
        // Translate elements with data-i18n-html (for HTML content)
        document.querySelectorAll('[data-i18n-html]').forEach(element => {
            const key = element.getAttribute('data-i18n-html');
            element.innerHTML = t(key);
        });
        
        // Translate title
        const titleElement = document.querySelector('[data-i18n-title]');
        if (titleElement) {
            document.title = t(titleElement.getAttribute('data-i18n-title'));
        }
    }
    
    /**
     * Initialize language selector dropdown
     */
    function initLanguageSelector() {
        const selector = document.getElementById('language-selector');
        if (selector) {
            // Set to current language
            selector.value = currentLanguage;
            
            // Add change listener
            selector.addEventListener('change', (e) => {
                setLanguage(e.target.value);
            });
        }
    }
    
    // Export to window
    window.i18n = {
        t: t,
        setLanguage: setLanguage,
        getCurrentLanguage: getCurrentLanguage,
        translatePage: translatePage,
        initLanguage: initLanguage,
        initLanguageSelector: initLanguageSelector,
        SUPPORTED_LANGUAGES: SUPPORTED_LANGUAGES
    };
    
    // Auto-initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => {
            initLanguage();
            initLanguageSelector();
            translatePage();
        });
    } else {
        initLanguage();
        initLanguageSelector();
        translatePage();
    }
})();
