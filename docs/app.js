/**
 * BramandScript Documentation Portal Client Script
 * Features: Dark/Light Mode, Copy-to-Clipboard, Live Search, Mobile Drawer, Scrollspy
 * Zero external JS dependencies.
 */

document.addEventListener('DOMContentLoaded', () => {
  initTheme();
  initCopyButtons();
  initSearch();
  initMobileMenu();
  initScrollspy();
});

// ---------------- 1. Theme Management (Dark / Light) ----------------
function initTheme() {
  const themeToggleBtn = document.getElementById('theme-toggle');
  
  // Check persisted preference or system preference
  const savedTheme = localStorage.getItem('bmd_docs_theme');
  const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  
  if (savedTheme === 'dark' || (!savedTheme && prefersDark)) {
    document.documentElement.classList.add('dark');
    updateThemeButtonUI(true);
  } else {
    document.documentElement.classList.remove('dark');
    updateThemeButtonUI(false);
  }

  if (themeToggleBtn) {
    themeToggleBtn.addEventListener('click', () => {
      const isDark = document.documentElement.classList.toggle('dark');
      localStorage.setItem('bmd_docs_theme', isDark ? 'dark' : 'light');
      updateThemeButtonUI(isDark);
    });
  }
}

function updateThemeButtonUI(isDark) {
  const themeToggleText = document.getElementById('theme-toggle-text');
  if (themeToggleText) {
    themeToggleText.textContent = isDark ? 'Light' : 'Dark';
  }
}

// ---------------- 2. Copy Code to Clipboard ----------------
function initCopyButtons() {
  const codeBlocks = document.querySelectorAll('.code-block-wrapper');

  codeBlocks.forEach(wrapper => {
    const copyBtn = wrapper.querySelector('.copy-btn');
    const pre = wrapper.querySelector('pre');
    
    if (copyBtn && pre) {
      copyBtn.addEventListener('click', async () => {
        const textToCopy = pre.innerText.trim();
        try {
          await navigator.clipboard.writeText(textToCopy);
          const originalHTML = copyBtn.innerHTML;
          copyBtn.classList.add('copied');
          copyBtn.innerHTML = `<svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg> <span>Copied!</span>`;
          setTimeout(() => {
            copyBtn.classList.remove('copied');
            copyBtn.innerHTML = originalHTML;
          }, 2000);
        } catch (err) {
          console.error('Failed to copy text: ', err);
        }
      });
    }
  });
}

// ---------------- 3. Live Search & Filter Bar ----------------
function initSearch() {
  const searchInput = document.getElementById('search-input');
  const navLinks = document.querySelectorAll('.nav-link');
  const searchResultsCount = document.getElementById('search-count');

  if (!searchInput) return;

  // Keyboard shortcut: '/' or 'Ctrl+K' / 'Cmd+K' to focus search
  window.addEventListener('keydown', (e) => {
    if ((e.key === '/' || ((e.ctrlKey || e.metaKey) && e.key === 'k')) && document.activeElement !== searchInput) {
      e.preventDefault();
      searchInput.focus();
      searchInput.select();
    } else if (e.key === 'Escape' && document.activeElement === searchInput) {
      searchInput.value = '';
      filterSections('');
      searchInput.blur();
    }
  });

  searchInput.addEventListener('input', (e) => {
    const query = e.target.value.toLowerCase().trim();
    filterSections(query);
  });

  function filterSections(query) {
    let matchedCount = 0;

    navLinks.forEach(link => {
      const text = link.textContent.toLowerCase();
      const parentLi = link.closest('li');
      if (!query || text.includes(query)) {
        if (parentLi) parentLi.style.display = '';
        matchedCount++;
      } else {
        if (parentLi) parentLi.style.display = 'none';
      }
    });

    if (searchResultsCount) {
      if (query) {
        searchResultsCount.textContent = `${matchedCount} matching`;
        searchResultsCount.classList.remove('hidden');
      } else {
        searchResultsCount.classList.add('hidden');
      }
    }
  }
}

// ---------------- 4. Mobile Drawer Navigation ----------------
function initMobileMenu() {
  const mobileMenuBtn = document.getElementById('mobile-menu-btn');
  const sidebar = document.getElementById('sidebar');
  const overlay = document.getElementById('sidebar-overlay');
  const navLinks = document.querySelectorAll('.nav-link');

  if (!mobileMenuBtn || !sidebar || !overlay) return;

  function openMenu() {
    sidebar.classList.remove('-translate-x-full');
    overlay.classList.remove('hidden');
    document.body.classList.add('overflow-hidden');
  }

  function closeMenu() {
    sidebar.classList.add('-translate-x-full');
    overlay.classList.add('hidden');
    document.body.classList.remove('overflow-hidden');
  }

  mobileMenuBtn.addEventListener('click', openMenu);
  overlay.addEventListener('click', closeMenu);

  navLinks.forEach(link => {
    link.addEventListener('click', () => {
      if (window.innerWidth < 1024) {
        closeMenu();
      }
    });
  });
}

// ---------------- 5. Scrollspy for Active Section ----------------
function initScrollspy() {
  const sections = document.querySelectorAll('section[id]');
  const navLinks = document.querySelectorAll('.nav-link');

  if (sections.length === 0 || navLinks.length === 0) return;

  const observerOptions = {
    root: null,
    rootMargin: '-80px 0px -70% 0px',
    threshold: 0
  };

  const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        const id = entry.target.getAttribute('id');
        setActiveNavLink(id);
      }
    });
  }, observerOptions);

  sections.forEach(section => observer.observe(section));

  function setActiveNavLink(id) {
    navLinks.forEach(link => {
      const href = link.getAttribute('href') || '';
      if (href === `#${id}` || href.endsWith(`#${id}`)) {
        link.classList.add('active');
        const sidebar = document.getElementById('sidebar-nav');
        if (sidebar) {
          const linkTop = link.offsetTop;
          const linkBottom = linkTop + link.offsetHeight;
          const sidebarScrollTop = sidebar.scrollTop;
          const sidebarHeight = sidebar.clientHeight;

          if (linkTop < sidebarScrollTop || linkBottom > (sidebarScrollTop + sidebarHeight)) {
            link.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
          }
        }
      } else {
        link.classList.remove('active');
      }
    });
  }
}
