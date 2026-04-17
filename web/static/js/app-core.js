(function () {
  'use strict';

  var app = window.ForumApp = window.ForumApp || {};

  app.state = {
    user: null,
    ws: null,
    currentFeedPage: 1,
    currentCategory: '',
    currentCategories: [],
    currentFilter: '',
    categories: [],
    pmOtherId: null,
    pmOtherName: '',
    pmOtherOnline: false,
    pmOldestId: null,
    pmHasMore: false,
    pmLoadMoreThrottled: null,
    pmLoading: false,
    pmOfflineNotice: null
  };

  app.els = {
    root: document.getElementById('root'),
    authScreen: document.getElementById('auth-screen'),
    appShell: document.getElementById('app-shell'),
    authError: document.getElementById('auth-error'),
    loginForm: document.getElementById('login-form'),
    registerForm: document.getElementById('register-form'),
    headerUsername: document.getElementById('header-username'),
    headerAvatarImg: document.getElementById('header-avatar-img'),
    headerAvatarFallback: document.getElementById('header-avatar-fallback'),
    avatarInput: document.getElementById('avatar-input'),
    avatarChangeBtn: document.getElementById('avatar-change-btn'),
    notifBtn: document.getElementById('notif-btn'),
    notifBadge: document.getElementById('notif-badge'),
    notifPanel: document.getElementById('notif-panel'),
    notifList: document.getElementById('notif-list'),
    notifReadAll: document.getElementById('notif-read-all'),
    mainContent: document.getElementById('main-content'),
    pageFeed: document.getElementById('page-feed'),
    pagePost: document.getElementById('page-post'),
    pageCreate: document.getElementById('page-create'),
    pageError: document.getElementById('page-error'),
    feedPosts: document.getElementById('feed-posts'),
    feedPagination: document.getElementById('feed-pagination'),
    categoryFilter: document.getElementById('category-filter'),
    postDetailContent: document.getElementById('post-detail-content'),
    postDetailComments: document.getElementById('post-detail-comments'),
    commentForm: document.getElementById('comment-form'),
    createPostForm: document.getElementById('create-post-form'),
    createPostCategories: document.getElementById('create-post-categories'),
    pmConversations: document.getElementById('pm-conversations'),
    pmChatPanel: document.getElementById('pm-chat-panel'),
    pmChatWithName: document.getElementById('pm-chat-with-name'),
    pmMessagesList: document.getElementById('pm-messages-list'),
    pmLoadMoreBtn: document.getElementById('pm-load-more-btn'),
    pmLoadMoreDiv: document.getElementById('pm-messages-load-more'),
    pmSendForm: document.getElementById('pm-send-form'),
    pmRecipientId: document.getElementById('pm-recipient-id'),
    pmMessagesContainer: document.getElementById('pm-messages'),
    pmChatClose: document.getElementById('pm-chat-close'),
    btnLogout: document.getElementById('btn-logout'),
    appErrorCode: document.getElementById('app-error-code'),
    appErrorTitle: document.getElementById('app-error-title'),
    appErrorMessage: document.getElementById('app-error-message'),
    appErrorHome: document.getElementById('app-error-home'),
    appErrorBack: document.getElementById('app-error-back')
  };

  app.show = function (el) { el.classList.remove('hidden'); };
  app.hide = function (el) { el.classList.add('hidden'); };
  app.showPage = function (page) {
    [app.els.pageFeed, app.els.pagePost, app.els.pageCreate, app.els.pageError].forEach(function (p) { app.hide(p); });
    app.show(page);
  };

  app.showErrorPage = function (cfg) {
    var data = cfg || {};
    var code = data.code || 'Ошибка';
    var title = data.title || 'Что-то пошло не так';
    var message = data.message || 'Попробуйте позже.';

    if (code === 400) {
      title = 'Плохой запрос';
      message = 'Боги не поняли твой призыв...';
    } else if (code === 404) {
      title = 'Страница пала в бою';
      message = 'Даже воины иногда теряют путь...';
    } else if (code === 500) {
      title = 'Сервер пал в Рагнарёке';
      message = 'Кузнецы уже чинят щиты...';
    }

    if (app.els.appErrorCode) app.els.appErrorCode.textContent = String(code);
    if (app.els.appErrorTitle) app.els.appErrorTitle.textContent = title;
    if (app.els.appErrorMessage) app.els.appErrorMessage.textContent = message;
    app.showPage(app.els.pageError);
  };

  app.categoryId = function (c) { return c && (c.id || c.ID); };
  app.categoryName = function (c) { return c && (c.name || c.Name) || ''; };

  app.normalizePath = function (pathname) {
    if (!pathname) return '/';
    var p = pathname;
    if (p.length > 1 && p.endsWith('/')) p = p.slice(0, -1);
    return p || '/';
  };

  app.navigate = function (path, replace) {
    var target = path || '/';
    if (window.location.pathname !== target) {
      if (replace) window.history.replaceState({}, '', target);
      else window.history.pushState({}, '', target);
    }
    if (typeof app.route === 'function') app.route();
  };

  app.fetchJSON = function (url, opts) {
    opts = opts || {};
    var headers = opts.headers || {};
    if (opts.json && !headers['Content-Type']) headers['Content-Type'] = 'application/json';
    return fetch(url, {
      method: opts.method || 'GET',
      headers: headers,
      body: opts.body,
      credentials: 'same-origin'
    }).then(function (res) {
      var contentType = (res.headers.get('content-type') || '').toLowerCase();
      if (contentType.indexOf('application/json') === -1) {
        if (!res.ok) {
          var e = new Error('HTTP ' + res.status);
          e.status = res.status;
          throw e;
        }
        return {};
      }
      return res.json().then(function (data) {
        if (res.ok) return data;
        var msg = (data && data.message) ? data.message : ('HTTP ' + res.status);
        var e = new Error(msg);
        e.status = res.status;
        throw e;
      });
    });
  };

  app.debounce = function (fn, ms) {
    var t;
    return function () {
      clearTimeout(t);
      t = setTimeout(fn, ms);
    };
  };

  app.throttle = function (fn, ms) {
    var last = 0;
    var timer = null;
    return function () {
      var now = Date.now();
      if (now - last >= ms) {
        last = now;
        fn();
      } else if (!timer) {
        timer = setTimeout(function () {
          timer = null;
          last = Date.now();
          fn();
        }, ms - (now - last));
      }
    };
  };

  app.escapeHtml = function (s) {
    if (!s) return '';
    var div = document.createElement('div');
    div.textContent = s;
    return div.innerHTML;
  };

  app.initials = function (name) {
    var n = (name || '').trim();
    if (!n) return '?';
    return n.slice(0, 1).toUpperCase();
  };

  app.avatarHTML = function (url, name, cls) {
    var className = cls || 'avatar-md';
    if (url) {
      return '<img class="avatar ' + className + '" src="' + app.escapeHtml(url) + '" alt="' + app.escapeHtml(name || 'avatar') + '">';
    }
    return '<div class="avatar avatar-fallback ' + className + '">' + app.escapeHtml(app.initials(name)) + '</div>';
  };

  app.renderHeaderAvatar = function () {
    var hasAvatar = app.state.user && app.state.user.avatar_url;
    if (!app.els.headerAvatarImg || !app.els.headerAvatarFallback) return;
    if (hasAvatar) {
      app.els.headerAvatarImg.src = app.state.user.avatar_url;
      app.els.headerAvatarImg.classList.remove('hidden');
      app.els.headerAvatarFallback.classList.add('hidden');
      return;
    }
    app.els.headerAvatarImg.classList.add('hidden');
    app.els.headerAvatarFallback.classList.remove('hidden');
    app.els.headerAvatarFallback.textContent = app.initials(app.state.user && app.state.user.nickname);
  };
})();
