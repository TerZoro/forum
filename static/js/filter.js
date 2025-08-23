// Simple post filtering - just highlight current sort option
document.addEventListener('DOMContentLoaded', function() {
    const urlParams = new URLSearchParams(window.location.search);
    const currentSort = urlParams.get('sort') || 'newest';

    const currentButton = document.querySelector(`[href="/?sort=${currentSort}"]`);
    if (currentButton) {
        currentButton.classList.add('is-active');
    }

    function applyMutualExclusion(btn) {
        if (btn.dataset.likeId) {
            const id = btn.dataset.likeId;
            document.querySelectorAll(`[data-like-id="${id}"]`).forEach(el => el.classList.add('is-active'));
            document.querySelectorAll(`[data-dislike-id="${id}"]`).forEach(el => el.classList.remove('is-active'));
            return true;
        }
        if (btn.dataset.dislikeId) {
            const id = btn.dataset.dislikeId;
            document.querySelectorAll(`[data-dislike-id="${id}"]`).forEach(el => el.classList.add('is-active'));
            document.querySelectorAll(`[data-like-id="${id}"]`).forEach(el => el.classList.remove('is-active'));
            return true;
        }
        if (btn.dataset.likeCommentId) {
            const id = btn.dataset.likeCommentId;
            document.querySelectorAll(`[data-like-comment-id="${id}"]`).forEach(el => el.classList.add('is-active'));
            document.querySelectorAll(`[data-dislike-comment-id="${id}"]`).forEach(el => el.classList.remove('is-active'));
            return true;
        }
        if (btn.dataset.dislikeCommentId) {
            const id = btn.dataset.dislikeCommentId;
            document.querySelectorAll(`[data-dislike-comment-id="${id}"]`).forEach(el => el.classList.add('is-active'));
            document.querySelectorAll(`[data-like-comment-id="${id}"]`).forEach(el => el.classList.remove('is-active'));
            return true;
        }
        return false;
    }

    document.body.addEventListener('pointerdown', function(e) {
        const btn = e.target.closest('button');
        if (!btn) return;
        applyMutualExclusion(btn);
    }, true);
});
