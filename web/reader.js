(() => {
  'use strict';
  const branches = [...document.querySelectorAll('.point > details')];
  const status = document.getElementById('status');
  const returnButton = document.getElementById('return-point');
  const previousPoints = [];

  function announce(message) {
    status.textContent = message;
  }

  function highlight(point) {
    if (!point) return;
    document.querySelectorAll('.point.highlight').forEach(item => item.classList.remove('highlight'));
    point.classList.add('highlight');
  }

  document.getElementById('outline').addEventListener('click', event => {
    highlight(event.target.closest('.point'));
  });

  document.getElementById('expand-all').addEventListener('click', () => {
    branches.forEach(branch => branch.open = true);
    announce('All points expanded');
  });
  document.getElementById('collapse-all').addEventListener('click', () => {
    branches.forEach(branch => branch.open = false);
    announce('Showing top-level points');
  });

  async function copyReference(value, button, message) {
    const title = button.title;
    let copied = false;
    // Clipboard permissions differ for file:// documents across browsers.
    if (navigator.clipboard?.writeText) {
      try { await navigator.clipboard.writeText(value); copied = true; } catch {}
    }
    if (!copied) {
      const field = document.createElement('textarea');
      field.value = value;
      field.style.cssText = 'position:fixed;left:-9999px;top:0;';
      document.body.append(field);
      field.select();
      try { copied = document.execCommand('copy'); } catch {}
      field.remove();
      button.focus({preventScroll:true});
    }
    if (copied) {
      button.classList.add('copied');
      button.title = message;
      announce(message);
      setTimeout(() => {
        button.classList.remove('copied');
        button.title = title;
      }, 1500);
    } else {
      const dialog = document.getElementById('copy-dialog');
      const field = document.getElementById('copy-value');
      field.value = value;
      field.rows = Math.min(12, value.split('\n').length);
      field.cols = Math.min(70, Math.max(12, ...value.split('\n').map(line => line.length)));
      dialog.showModal();
      field.focus();
      field.select();
    }
  }

  document.querySelectorAll('.copy-point, .copy-markdown').forEach(button => {
    button.addEventListener('click', event => {
      event.preventDefault();
      event.stopPropagation();
      const point = button.closest('.point');
      highlight(point);
      const isMarkdown = button.classList.contains('copy-markdown');
      const value = isMarkdown ? button.dataset.markdown : button.dataset.copy;
      const message = isMarkdown ? `Copied Markdown for [${point.id.slice(2)}]` : `Copied ${value}`;
      void copyReference(value, button, message);
    });
    button.addEventListener('keydown', event => event.stopPropagation());
  });

  function reveal(hash, focus = true) {
    const point = document.getElementById(hash.replace(/^#/, ''));
    if (!point?.classList.contains('point')) return;
    // Open ancestors, keeping the target's own children collapsed as they were.
    for (let parent = point.parentElement; parent; parent = parent.parentElement) {
      if (parent.tagName === 'DETAILS') parent.open = true;
    }
    highlight(point);
    requestAnimationFrame(() => {
      point.scrollIntoView({block:'center', behavior:'instant'});
      if (focus) {
        const row = point.querySelector(':scope > details > summary, :scope > .leaf-row');
        if (row.tagName !== 'SUMMARY') row.tabIndex = -1;
        row.focus({preventScroll:true});
      }
    });
  }

  document.querySelectorAll('a.xref').forEach(link => {
    link.addEventListener('click', event => {
      if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return;
      event.preventDefault();
      event.stopPropagation();
      const origin = link.closest('.point');
      if (origin) previousPoints.push(origin.id);
      returnButton.hidden = previousPoints.length === 0;
      const hash = link.getAttribute('href');
      if (location.hash !== hash) location.hash = hash;
      reveal(hash);
    });
  });

  returnButton.addEventListener('click', () => {
    const id = previousPoints.pop();
    returnButton.hidden = previousPoints.length === 0;
    if (id) { location.hash = id; reveal('#' + id); }
  });

  document.querySelectorAll('.image-size').forEach(button => {
    button.addEventListener('click', () => {
      const image = button.closest('.image-block');
      const full = image.classList.toggle('full-size');
      button.textContent = full ? 'Small size' : 'Full size';
      button.setAttribute('aria-pressed', String(full));
    });
  });

  if (document.querySelector('.diagram')) {
    let peleLoading;
    function loadPele() {
      if (window.pele) return Promise.resolve();
      if (!peleLoading) {
        peleLoading = new Promise((resolve, reject) => {
          const script = document.createElement('script');
          script.src = 'https://cdn.jsdelivr.net/npm/pele@1.13.0/dist/pele.global.js';
          script.integrity = 'sha384-vDPV1LDXGIwjU5/gMUtHbn/WAmwRmOgkQpiq+Rm4d/BPTsdfpIUpvi2xVET6z85W';
          script.crossOrigin = 'anonymous';
          script.onload = () => window.pele ? resolve() : reject(new Error('Pele did not initialize.'));
          script.onerror = () => reject(new Error('Pele could not be loaded from the CDN; an internet connection is required.'));
          document.head.append(script);
        });
      }
      return peleLoading;
    }
    async function renderVisibleDiagrams() {
      for (const diagram of document.querySelectorAll('.diagram:not([data-rendered])')) {
        let hidden = false;
        for (let parent = diagram.parentElement; parent; parent = parent.parentElement) {
          if (parent.tagName === 'DETAILS' && !parent.open) { hidden = true; break; }
        }
        if (hidden) continue;
        const bounds = diagram.getBoundingClientRect();
        if (!bounds.width || !bounds.height) continue;
        diagram.dataset.rendered = 'pending';
        const source = diagram.querySelector('pre').textContent;
        try {
          await loadPele();
          const mounted = window.pele.mount(diagram, source, {links:false, images:false, zoom:false});
          if (mounted.error) throw mounted.error;
          diagram.dataset.rendered = 'true';
        } catch (error) {
          diagram.dataset.rendered = 'error';
          const message = document.createElement('p');
          message.textContent = `Diagram could not be rendered: ${error.message}`;
          diagram.append(message);
        }
      }
    }
    document.addEventListener('toggle', event => {
      if (event.target.tagName === 'DETAILS' && event.target.open) {
        requestAnimationFrame(() => void renderVisibleDiagrams());
      }
    }, true);
    requestAnimationFrame(() => void renderVisibleDiagrams());
  }
  window.addEventListener('hashchange', () => reveal(location.hash));
  if (location.hash) reveal(location.hash);
})();
