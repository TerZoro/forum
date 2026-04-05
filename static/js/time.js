document.querySelectorAll('[data-utc]').forEach(function(el) {
    var d = new Date(el.getAttribute('data-utc'));
    if (isNaN(d)) return;
    var opts = { month: 'short', day: '2-digit', year: 'numeric' };
    if (el.getAttribute('data-fmt') === 'datetime') {
        opts.hour = '2-digit';
        opts.minute = '2-digit';
        opts.hour12 = false;
    }
    el.textContent = d.toLocaleString(undefined, opts);
});
