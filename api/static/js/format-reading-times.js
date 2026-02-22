(function () {
  var formatter = new Intl.DateTimeFormat(undefined, { timeStyle: 'short', hour12: false });
  function formatReadingTimes(root) {
    (root || document).querySelectorAll('.reading-time[data-time]').forEach(function (el) {
      var date = new Date(el.getAttribute('data-time'));
      if (!isNaN(date.getTime())) {
        el.textContent = 'Updated ' + formatter.format(date);
      }
    });
  }
  formatReadingTimes();
  document.body.addEventListener('htmx:afterSwap', function (ev) {
    if (ev.detail.target.id === 'stations-container') formatReadingTimes(ev.detail.target);
  });
})();
