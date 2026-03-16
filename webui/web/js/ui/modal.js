/**
 * Simple modal dialog component.
 * Shows a centered card with title, message, and configurable buttons.
 * Returns a Promise that resolves with the clicked button's id.
 *
 * Usage:
 *   const result = await showModal({
 *       title: 'Device Unreachable',
 *       message: 'This IP is not responding.',
 *       buttons: [
 *           { id: 'change', label: 'Change IP', style: 'primary' },
 *           { id: 'continue', label: 'Continue Anyway', style: 'outline' }
 *       ]
 *   });
 */

let currentResolve = null;

export function showModal({ title, message, buttons }) {
    return new Promise((resolve) => {
        currentResolve = resolve;

        const overlay = document.getElementById('modal-overlay');

        // Clear previous content safely
        overlay.replaceChildren();

        // Build modal DOM using safe DOM methods
        const modal = document.createElement('div');
        modal.className = 'modal';

        const titleEl = document.createElement('div');
        titleEl.className = 'modal-title';
        titleEl.textContent = title;
        modal.appendChild(titleEl);

        const messageEl = document.createElement('div');
        messageEl.className = 'modal-message';
        messageEl.textContent = message;
        modal.appendChild(messageEl);

        const actionsEl = document.createElement('div');
        actionsEl.className = 'modal-actions';

        buttons.forEach(btnConfig => {
            const btn = document.createElement('button');
            btn.className = `btn btn-${btnConfig.style || 'outline'}`;
            btn.textContent = btnConfig.label;
            btn.addEventListener('click', () => {
                hideModal();
                resolve(btnConfig.id);
            });
            actionsEl.appendChild(btn);
        });

        modal.appendChild(actionsEl);
        overlay.appendChild(modal);

        // Close on overlay click (outside modal)
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                hideModal();
                resolve(null);
            }
        });

        // Show with animation
        overlay.classList.remove('hidden');
        requestAnimationFrame(() => {
            overlay.classList.add('show');
        });
    });
}

export function hideModal() {
    const overlay = document.getElementById('modal-overlay');
    overlay.classList.remove('show');
    setTimeout(() => {
        overlay.classList.add('hidden');
        overlay.replaceChildren();
    }, 200);
    currentResolve = null;
}
