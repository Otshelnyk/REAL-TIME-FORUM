(function () {
  'use strict';

  var app = window.ForumApp;
  var state = app.state;
  var els = app.els;
  var notifPollTimer = null;

  function renderEmptyState() {
    if (!els.notifList) return;
    if (els.notifPanel) els.notifPanel.classList.add('no-items');
    els.notifList.innerHTML =
      '<div class="notif-empty-state">' +
      '<div class="notif-empty-icon">🔔</div>' +
      '<div class="notif-empty-title">No notifications yet</div>' +
      '<div class="notif-empty-subtitle">New likes, comments and messages will appear here.</div>' +
      '</div>';
  }

  function renderNotifications(items, unreadCount) {
    if (!els.notifBadge || !els.notifList) return;
    if (unreadCount > 0) {
      els.notifBadge.textContent = String(unreadCount > 99 ? '99+' : unreadCount);
      els.notifBadge.classList.remove('hidden');
    } else {
      els.notifBadge.classList.add('hidden');
    }

    if (!items || !items.length) {
      renderEmptyState();
      return;
    }
    if (els.notifPanel) els.notifPanel.classList.remove('no-items');

    els.notifList.innerHTML = items.map(function (n) {
      var unreadClass = n.is_read ? '' : ' unread';
      return (
        '<button type="button" class="notif-item' + unreadClass + '" data-id="' + n.id + '" data-link="' + app.escapeHtml(n.link || '') + '">' +
        '<div class="notif-title">' + app.escapeHtml(n.title || '') + '</div>' +
        '<div class="notif-message">' + app.escapeHtml(n.message || '') + '</div>' +
        '<div class="notif-time">' + app.escapeHtml(n.created_at || '') + '</div>' +
        '</button>'
      );
    }).join('');

    els.notifList.querySelectorAll('.notif-item').forEach(function (btn) {
      btn.addEventListener('click', function () {
        var id = parseInt(btn.dataset.id, 10);
        var link = btn.dataset.link || '';
        app.fetchJSON('/api/notifications/read', {
          method: 'POST',
          json: true,
          body: JSON.stringify({ id: id })
        }).finally(function () {
          app.loadNotifications();
          if (link) app.navigate(link);
          if (els.notifPanel) els.notifPanel.classList.add('hidden');
        });
      });
    });
  }

  app.loadNotifications = function () {
    if (!state.user) return;
    app.fetchJSON('/api/notifications?limit=30').then(function (data) {
      if (!data.success || !data.data) return;
      renderNotifications(data.data.items || [], data.data.unread_count || 0);
    });
  };

  function startPolling() {
    if (notifPollTimer) return;
    notifPollTimer = setInterval(function () {
      if (!state.user) return;
      app.loadNotifications();
    }, 2500);
  }

  function stopPolling() {
    if (!notifPollTimer) return;
    clearInterval(notifPollTimer);
    notifPollTimer = null;
  }

  app.startNotificationsPolling = startPolling;
  app.stopNotificationsPolling = stopPolling;

  app.resetNotifications = function () {
    stopPolling();
    if (els.notifBadge) els.notifBadge.classList.add('hidden');
    renderEmptyState();
    if (els.notifPanel) els.notifPanel.classList.add('hidden');
  };

  app.bindNotificationsEvents = function () {
    if (!els.notifBtn || !els.notifPanel || !els.notifReadAll) return;
    els.notifBtn.addEventListener('click', function (e) {
      e.stopPropagation();
      els.notifPanel.classList.toggle('hidden');
      if (!els.notifPanel.classList.contains('hidden')) {
        app.loadNotifications();
      }
    });

    els.notifPanel.addEventListener('click', function (e) {
      e.stopPropagation();
    });

    document.addEventListener('click', function () {
      els.notifPanel.classList.add('hidden');
    });

    els.notifReadAll.addEventListener('click', function () {
      app.fetchJSON('/api/notifications/read-all', { method: 'POST' }).then(function () {
        app.loadNotifications();
      });
    });

    document.addEventListener('visibilitychange', function () {
      if (!state.user) return;
      if (document.visibilityState === 'visible') app.loadNotifications();
    });

    startPolling();
  };
})();
