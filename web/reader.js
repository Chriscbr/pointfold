(() => {
  'use strict';
  const originalPage = '<!doctype html>\n' + document.documentElement.outerHTML;
  const branches = [...document.querySelectorAll('.point > details')];
  const status = document.getElementById('status');
  const returnButton = document.getElementById('return-point');
  const previousPoints = [];
  let renderDiagrams = async () => {};

  const themeButton = document.getElementById('theme-toggle');
  function updateThemeButton() {
    const next = document.documentElement.dataset.theme === 'dark' ? 'Light' : 'Dark';
    themeButton.textContent = `${next} mode`;
    themeButton.setAttribute('aria-label', `Switch to ${next.toLowerCase()} mode`);
  }
  themeButton.addEventListener('click', () => {
    const theme = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
    document.documentElement.dataset.theme = theme;
    try { localStorage.setItem('pointfold-theme', theme); } catch {}
    updateThemeButton();
  });
  updateThemeButton();

  function announce(message) {
    status.textContent = message;
  }

  function saveFile(blob, filename) {
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.append(link);
    link.click();
    link.remove();
    setTimeout(() => URL.revokeObjectURL(url), 30000);
  }

  function decodeBase64(value) {
    return Uint8Array.from(atob(value), char => char.charCodeAt(0));
  }

  const exportData = JSON.parse(document.getElementById('export-data').textContent);
  const filename = exportData.Filename.replace(/[\\/:*?"<>|\x00-\x1f]/g, '-') || 'outline.md';
  const stem = filename.replace(/\.[^.]+$/, '') || 'outline';
  document.getElementById('download-page').addEventListener('click', () => {
    saveFile(new Blob([originalPage], {type:'text/html;charset=utf-8'}), `${stem}.html`);
  });
  const sourceBytes = decodeBase64(exportData.Source);
  const fullMarkdown = new TextDecoder('utf-8', {ignoreBOM:true}).decode(sourceBytes);
  document.getElementById('download-all').addEventListener('click', () => {
    saveFile(new Blob([sourceBytes], {type:'text/markdown;charset=utf-8'}), `${stem}.md`);
  });
  document.getElementById('copy-all').addEventListener('click', event => {
    void copyText(fullMarkdown, event.currentTarget, 'Copied complete Markdown');
  });

  let printState;
  function preparePrint() {
    if (!printState) printState = branches.map(branch => branch.open);
    branches.forEach(branch => branch.open = true);
  }
  function restorePrint() {
    if (!printState) return;
    branches.forEach((branch, index) => branch.open = printState[index]);
    printState = undefined;
  }
  window.addEventListener('beforeprint', preparePrint);
  window.addEventListener('afterprint', restorePrint);
  document.getElementById('print-page').addEventListener('click', async () => {
    const button = document.getElementById('print-page');
    button.disabled = true;
    preparePrint();
    try {
      await renderDiagrams();
      await Promise.all([...document.querySelectorAll('img')].map(image => image.decode().catch(() => {})));
      await document.fonts.ready;
      window.print();
    } finally {
      restorePrint();
      button.disabled = false;
    }
  });

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

  async function copyText(value, button, message) {
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
      field.cols = Math.min(70, value.split('\n').reduce((width, line) => Math.max(width, line.length), 12));
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
      void copyText(value, button, message);
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
    const rendering = new Set();
    renderDiagrams = () => {
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
        const task = (async () => {
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
        })();
        rendering.add(task);
        void task.finally(() => rendering.delete(task));
      }
      return Promise.all([...rendering]);
    };
    document.addEventListener('toggle', event => {
      if (event.target.tagName === 'DETAILS' && event.target.open) {
        requestAnimationFrame(() => void renderDiagrams());
      }
    }, true);
    requestAnimationFrame(() => void renderDiagrams());
  }
  window.addEventListener('hashchange', () => reveal(location.hash));
  if (location.hash) reveal(location.hash);
})();
