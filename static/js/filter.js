// Simple post filtering - just highlight current sort option
document.addEventListener('DOMContentLoaded', function() {
    // Get current sort from URL
    const urlParams = new URLSearchParams(window.location.search);
    const currentSort = urlParams.get('sort') || 'newest';
    
    // Find the current sort button and highlight it
    const currentButton = document.querySelector(`[href="/?sort=${currentSort}"]`);
    if (currentButton) {
        currentButton.style.opacity = '0.7';
        currentButton.style.transform = 'scale(0.95)';
    }
});
