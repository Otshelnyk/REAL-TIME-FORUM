(function () {
  'use strict';

  var app = window.ForumApp;
  var state = app.state;
  var els = app.els;
  var typingStopTimer = null;
  var remoteTypingTimer = null;
  var typingByUser = {};
  var chatDrag = {
    active: false,
    offsetX: 0,
    offsetY: 0
  };
  var chatPositionCustomized = false;

  function renderConversationAvatar(user) {
    if (typeof app.avatarHTML === 'function') {
      return app.avatarHTML(user.avatar_url, user.nickname, 'avatar-xs');
    }
    var letter = (user.nickname || '?').trim().slice(0, 1).toUpperCase() || '?';
    if (user.avatar_url) {
      return '<img class="avatar avatar-xs" src="' + app.escapeHtml(user.avatar_url) + '" alt="' + app.escapeHtml(user.nickname || 'avatar') + '">';
    }
    return '<div class="avatar avatar-fallback avatar-xs">' + app.escapeHtml(letter) + '</div>';
  }

  function updatePMAvailability() {
    var input = els.pmSendForm.querySelector('input[name="content"]');
    var sendBtn = els.pmSendForm.querySelector('button[type="submit"]');
    if (!state.pmOfflineNotice) {
      state.pmOfflineNotice = document.createElement('div');
      state.pmOfflineNotice.className = 'auth-hint';
      state.pmOfflineNotice.style.marginTop = '8px';
      els.pmSendForm.appendChild(state.pmOfflineNotice);
    }
    if (!state.pmOtherOnline) {
      input.disabled = true;
      sendBtn.disabled = true;
      state.pmOfflineNotice.textContent = 'User is offline. Sending is disabled.';
      state.pmOfflineNotice.classList.remove('hidden');
      return;
    }
    input.disabled = false;
    sendBtn.disabled = false;
    state.pmOfflineNotice.textContent = '';
    state.pmOfflineNotice.classList.add('hidden');
  }

  function ensureTypingIndicator() {
    var indicator = els.pmMessagesList.querySelector('.typing-indicator');
    if (indicator) return indicator;
    indicator = document.createElement('div');
    indicator.className = 'typing-indicator hidden';
    indicator.innerHTML = '<span class="dot"></span><span class="dot"></span><span class="dot"></span><span class="typing-text">typing...</span>';
    els.pmMessagesList.appendChild(indicator);
    return indicator;
  }

  function showRemoteTyping(show) {
    var indicator = ensureTypingIndicator();
    if (!indicator) return;
    if (show) {
      indicator.classList.remove('hidden');
      autoScrollToBottom(true);
      return;
    }
    indicator.classList.add('hidden');
  }

  function scrollToBottom() {
    els.pmMessagesContainer.scrollTop = els.pmMessagesContainer.scrollHeight;
  }

  function isNearBottom() {
    var threshold = 80;
    var fromBottom = els.pmMessagesContainer.scrollHeight - (els.pmMessagesContainer.scrollTop + els.pmMessagesContainer.clientHeight);
    return fromBottom <= threshold;
  }

  function autoScrollToBottom(force) {
    if (force || isNearBottom()) {
      scrollToBottom();
    }
  }

  function appendIncomingMessage(payload) {
    var inner = els.pmMessagesList.querySelector('.pm-messages-list-inner');
    if (!inner) return;
    var div = document.createElement('div');
    div.className = 'pm-msg';
    div.innerHTML = '<div class="meta">' + app.escapeHtml(payload.from_nickname || '') + ' • ' + app.escapeHtml(payload.created_at || '') + '</div><div class="content">' + app.escapeHtml(payload.content || '') + '</div>';
    inner.appendChild(div);
    autoScrollToBottom(false);
  }

  function clampChatPosition(left, top) {
    var panel = els.pmChatPanel;
    var maxLeft = Math.max(0, window.innerWidth - panel.offsetWidth);
    var maxTop = Math.max(0, window.innerHeight - panel.offsetHeight);
    return {
      left: Math.min(Math.max(0, left), maxLeft),
      top: Math.min(Math.max(0, top), maxTop)
    };
  }

  function setChatPosition(left, top) {
    var pos = clampChatPosition(left, top);
    els.pmChatPanel.style.left = pos.left + 'px';
    els.pmChatPanel.style.top = pos.top + 'px';
    els.pmChatPanel.style.right = 'auto';
    els.pmChatPanel.style.bottom = 'auto';
  }

  function resetChatPosition() {
    chatPositionCustomized = false;
    els.pmChatPanel.style.left = '';
    els.pmChatPanel.style.top = '';
    els.pmChatPanel.style.right = '';
    els.pmChatPanel.style.bottom = '';
  }

  function enableChatDragging() {
    if (!els.pmChatPanel) return;
    var header = els.pmChatPanel.querySelector('.pm-chat-header');
    if (!header) return;

    function getPoint(event) {
      if (event.touches && event.touches.length) return event.touches[0];
      return event;
    }

    function onMove(event) {
      if (!chatDrag.active) return;
      var point = getPoint(event);
      setChatPosition(point.clientX - chatDrag.offsetX, point.clientY - chatDrag.offsetY);
      if (event.cancelable) event.preventDefault();
    }

    function onEnd() {
      if (!chatDrag.active) return;
      chatDrag.active = false;
      document.body.classList.remove('pm-chat-dragging');
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onEnd);
      window.removeEventListener('touchmove', onMove);
      window.removeEventListener('touchend', onEnd);
      window.removeEventListener('touchcancel', onEnd);
    }

    function onStart(event) {
      if (event.target && event.target.closest('.pm-chat-header-btn')) return;
      var point = getPoint(event);
      var rect = els.pmChatPanel.getBoundingClientRect();
      chatDrag.active = true;
      chatPositionCustomized = true;
      chatDrag.offsetX = point.clientX - rect.left;
      chatDrag.offsetY = point.clientY - rect.top;
      setChatPosition(rect.left, rect.top);
      document.body.classList.add('pm-chat-dragging');
      window.addEventListener('mousemove', onMove);
      window.addEventListener('mouseup', onEnd);
      window.addEventListener('touchmove', onMove, { passive: false });
      window.addEventListener('touchend', onEnd);
      window.addEventListener('touchcancel', onEnd);
      if (event.cancelable) event.preventDefault();
    }

    header.addEventListener('mousedown', onStart);
    header.addEventListener('touchstart', onStart, { passive: false });

    window.addEventListener('resize', function () {
      if (els.pmChatPanel.classList.contains('hidden')) return;
      if (!chatPositionCustomized) {
        resetChatPosition();
        return;
      }
      var rect = els.pmChatPanel.getBoundingClientRect();
      setChatPosition(rect.left, rect.top);
    });
  }

  function openChat(otherId, otherName, isOnline) {
    state.pmOtherId = otherId;
    state.pmOtherName = otherName;
    state.pmOtherOnline = !!isOnline;
    state.pmOldestId = null;
    state.pmHasMore = false;
    els.pmRecipientId.value = otherId;
    els.pmChatWithName.textContent = otherName;
    els.pmConversations.querySelectorAll('.pm-conv-item').forEach(function (a) {
      a.classList.toggle('active', parseInt(a.dataset.userId, 10) === otherId);
    });
    app.show(els.pmChatPanel);
    if (!chatPositionCustomized) resetChatPosition();
    updatePMAvailability();
    els.pmMessagesList.innerHTML = '';
    showRemoteTyping(typingByUser[otherId] === true);
    app.show(els.pmLoadMoreDiv);
    app.loadPMMessages();
  }

  function renderConversationItem(u) {
    var online = u.online ? '<span class="online-dot"></span>' : '<span class="online-dot offline"></span>';
    var isTyping = typingByUser[u.user_id] === true;
    var preview = isTyping
      ? '<div class="preview typing-preview"><span class="typing-dots"><span>.</span><span>.</span><span>.</span></span> typing...</div>'
      : (u.last_msg_preview ? '<div class="preview">' + app.escapeHtml(u.last_msg_preview) + '</div>' : '');
    return (
      '<a href="#" class="pm-conv-item' + (state.pmOtherId === u.user_id ? ' active' : '') + '" data-user-id="' + u.user_id + '" data-online="' + (u.online ? '1' : '0') + '" data-nickname="' + app.escapeHtml(u.nickname) + '">' +
      renderConversationAvatar(u) + online + '<span class="nickname">' + app.escapeHtml(u.nickname) + '</span>' + preview + '</a>'
    );
  }

  app.loadConversations = function () {
    app.fetchJSON('/api/messages/conversations').then(function (data) {
      if (!data.success || !data.data) return;
      var onlineUsers = [];
      var offlineUsers = [];
      data.data.forEach(function (u) {
        if (u.online) onlineUsers.push(u);
        else offlineUsers.push(u);
      });

      var html = '';
      if (onlineUsers.length) {
        html += '<div class="pm-conv-group"><div class="pm-conv-group-title">Online</div>' +
          onlineUsers.map(renderConversationItem).join('') +
          '</div>';
      }
      if (offlineUsers.length) {
        html += '<div class="pm-conv-group"><div class="pm-conv-group-title">Offline</div>' +
          offlineUsers.map(renderConversationItem).join('') +
          '</div>';
      }
      if (!html) html = '<div class="pm-conv-empty">No users available</div>';
      els.pmConversations.innerHTML = html;
      els.pmConversations.querySelectorAll('.pm-conv-item').forEach(function (a) {
        a.addEventListener('click', function (e) {
          e.preventDefault();
          openChat(parseInt(a.dataset.userId, 10), a.dataset.nickname, a.dataset.online === '1');
        });
      });
      if (state.pmOtherId) {
        var active = els.pmConversations.querySelector('.pm-conv-item[data-user-id="' + state.pmOtherId + '"]');
        if (active) {
          state.pmOtherOnline = active.dataset.online === '1';
          updatePMAvailability();
        }
      }
    });
  };

  app.loadPMMessages = function (beforeId) {
    if (!state.pmOtherId || state.pmLoading) return;
    state.pmLoading = true;
    var url = '/api/messages/with/' + state.pmOtherId + '?limit=10';
    if (beforeId) url += '&before=' + beforeId;
    app.fetchJSON(url).then(function (data) {
      state.pmLoading = false;
      if (!data.success || !data.data) return;
      var list = data.data.messages || [];
      state.pmHasMore = data.data.has_more;
      if (list.length) state.pmOldestId = list[0].id;
      var inner = els.pmMessagesList.querySelector('.pm-messages-list-inner') || (function () {
        var w = document.createElement('div');
        w.className = 'pm-messages-list-inner';
        els.pmMessagesList.appendChild(w);
        return w;
      })();
      if (beforeId) {
        for (var i = list.length - 1; i >= 0; i--) {
          var m = list[i];
          var div = document.createElement('div');
          div.className = 'pm-msg';
          div.dataset.msgId = m.id;
          div.innerHTML = '<div class="meta">' + app.escapeHtml(m.from_nickname) + ' • ' + app.escapeHtml(m.created_at) + '</div><div class="content">' + app.escapeHtml(m.content) + '</div>';
          inner.insertBefore(div, inner.firstChild);
        }
      } else {
        list.forEach(function (m) {
          var div = document.createElement('div');
          div.className = 'pm-msg';
          div.dataset.msgId = m.id;
          div.innerHTML = '<div class="meta">' + app.escapeHtml(m.from_nickname) + ' • ' + app.escapeHtml(m.created_at) + '</div><div class="content">' + app.escapeHtml(m.content) + '</div>';
          inner.appendChild(div);
        });
      }
      if (!state.pmHasMore) app.hide(els.pmLoadMoreDiv);
      else {
        app.show(els.pmLoadMoreDiv);
        els.pmLoadMoreBtn.textContent = 'Load older messages';
      }
      if (!beforeId) {
        setTimeout(function () { scrollToBottom(); }, 0);
      }
    }).catch(function () { state.pmLoading = false; });
  };

  app.connectWS = function () {
    var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    state.ws = new WebSocket(protocol + '//' + window.location.host + '/ws');
    state.ws.onmessage = function (ev) {
      try {
        var msg = JSON.parse(ev.data);
        if (msg.type === 'new_private_message' && msg.payload && msg.payload.from_id === state.pmOtherId) {
          appendIncomingMessage(msg.payload);
          typingByUser[msg.payload.from_id] = false;
          showRemoteTyping(false);
          app.loadConversations();
          if (typeof app.loadNotifications === 'function') app.loadNotifications();
        } else if (msg.type === 'new_private_message' && msg.payload && msg.payload.from_id) {
          typingByUser[msg.payload.from_id] = false;
          app.loadConversations();
          if (typeof app.loadNotifications === 'function') app.loadNotifications();
        } else if (msg.type === 'notification_created') {
          if (typeof app.loadNotifications === 'function') app.loadNotifications();
        } else if (msg.type === 'user_presence' && msg.payload) {
          app.loadConversations();
        } else if (msg.type === 'typing' && msg.payload && msg.payload.from_id) {
          typingByUser[msg.payload.from_id] = !!msg.payload.is_typing;
          if (msg.payload.from_id === state.pmOtherId) {
            showRemoteTyping(!!msg.payload.is_typing);
          }
          app.loadConversations();
          if (remoteTypingTimer) clearTimeout(remoteTypingTimer);
          if (msg.payload.is_typing) {
            remoteTypingTimer = setTimeout(function () {
              typingByUser[msg.payload.from_id] = false;
              showRemoteTyping(false);
              app.loadConversations();
            }, 2500);
          }
        }
      } catch (e) {}
    };
    state.ws.onclose = function () {
      setTimeout(function () { if (state.user) app.connectWS(); }, 2000);
    };
  };

  app.bindPMEvents = function () {
    enableChatDragging();
    var resetBtn = els.pmChatPanel.querySelector('#pm-chat-reset');

    els.pmChatClose.addEventListener('click', function () {
      app.hide(els.pmChatPanel);
      state.pmOtherId = null;
      state.pmOtherOnline = false;
    });

    if (resetBtn) {
      resetBtn.addEventListener('click', function () {
        resetChatPosition();
      });
    }

    state.pmLoadMoreThrottled = app.throttle(function () {
      if (!state.pmOtherId || !state.pmHasMore || !state.pmOldestId || state.pmLoading) return;
      app.loadPMMessages(state.pmOldestId);
    }, 400);

    els.pmMessagesContainer.addEventListener('scroll', function () {
      if (this.scrollTop <= 0) state.pmLoadMoreThrottled();
    });

    els.pmLoadMoreBtn.addEventListener('click', function () {
      if (state.pmOldestId && state.pmHasMore) app.loadPMMessages(state.pmOldestId);
    });

    els.pmSendForm.addEventListener('submit', function (e) {
      e.preventDefault();
      var content = this.content.value.trim();
      if (!content || !state.pmOtherId || !state.pmOtherOnline || !state.ws || state.ws.readyState !== 1) return;
      if (typingStopTimer) {
        clearTimeout(typingStopTimer);
        typingStopTimer = null;
      }
      state.ws.send(JSON.stringify({
        type: 'typing',
        payload: { to_id: state.pmOtherId, is_typing: false }
      }));
      state.ws.send(JSON.stringify({
        type: 'private_message',
        payload: { to_id: state.pmOtherId, content: content }
      }));
      this.content.value = '';
      var inner = els.pmMessagesList.querySelector('.pm-messages-list-inner');
      if (inner) {
        var div = document.createElement('div');
        div.className = 'pm-msg';
        div.innerHTML = '<div class="meta">' + app.escapeHtml(state.user.nickname) + ' • now</div><div class="content">' + app.escapeHtml(content) + '</div>';
        inner.appendChild(div);
        autoScrollToBottom(true);
      }
    });

    var input = els.pmSendForm.querySelector('input[name="content"]');
    input.addEventListener('input', function () {
      if (!state.pmOtherId || !state.ws || state.ws.readyState !== 1) return;
      state.ws.send(JSON.stringify({
        type: 'typing',
        payload: { to_id: state.pmOtherId, is_typing: true }
      }));
      if (typingStopTimer) clearTimeout(typingStopTimer);
      typingStopTimer = setTimeout(function () {
        if (!state.pmOtherId || !state.ws || state.ws.readyState !== 1) return;
        state.ws.send(JSON.stringify({
          type: 'typing',
          payload: { to_id: state.pmOtherId, is_typing: false }
        }));
      }, 900);
    });

    input.addEventListener('blur', function () {
      if (!state.pmOtherId || !state.ws || state.ws.readyState !== 1) return;
      state.ws.send(JSON.stringify({
        type: 'typing',
        payload: { to_id: state.pmOtherId, is_typing: false }
      }));
    });
  };
})();
