// PWA: registro del service worker y botón "Instalar aplicación".
// El botón permanece oculto hasta que el navegador ofrece la instalación
// (beforeinstallprompt) y desaparece al instalarse o si ya se ejecuta como app.

function isStandalone() {
  try {
    return window.matchMedia('(display-mode: standalone)').matches;
  } catch (_) {
    return false;
  }
}

export function registerServiceWorker() {
  if (!('serviceWorker' in navigator)) return;
  navigator.serviceWorker.register('/sw.js', { scope: '/' }).catch(err => {
    console.warn('No se pudo registrar el service worker:', err);
  });
}

export function initInstallButton() {
  const btn = document.getElementById('btn-install-app');
  if (!btn) return;
  let deferred = null;

  if (isStandalone()) {
    btn.hidden = true;
    return;
  }

  window.addEventListener('beforeinstallprompt', event => {
    event.preventDefault();
    deferred = event;
    btn.hidden = false;
  });

  btn.addEventListener('click', async () => {
    if (!deferred) return;
    const prompt = deferred;
    deferred = null;
    btn.hidden = true;
    try {
      await prompt.prompt();
    } catch (err) {
      console.warn('No se pudo mostrar el aviso de instalación:', err);
    }
  });

  window.addEventListener('appinstalled', () => {
    deferred = null;
    btn.hidden = true;
  });
}
