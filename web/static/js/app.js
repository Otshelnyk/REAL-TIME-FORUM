(function () {
  'use strict';

  var app = window.ForumApp;
  var state = app.state;

  app.route = function () {
    var path = app.normalizePath(window.location.pathname);
    var parts = path.split('/');
    var page = parts[1] || 'feed';

    if (page === 'error') {
      var errCfg = (typeof app.getErrorFromLocation === 'function') ? app.getErrorFromLocation() : null;
      app.showErrorPage(errCfg || { code: 500 });
      return;
    }

    if (!state.user) {
      if (page === 'register') {
        app.showRegisterForm();
        return;
      }
      app.showLoginForm();
      return;
    }

    if (page === 'login' || page === 'register') {
      app.navigate('/', true);
      return;
    }

    if (path === '/' || page === 'feed') {
      app.showPage(app.els.pageFeed);
      state.currentFeedPage = parseInt(parts[2], 10) || 1;
      app.loadFeed();
      return;
    }

    if (page === 'post' && parts[2]) {
      app.showPage(app.els.pagePost);
      app.loadPostDetail(parseInt(parts[2], 10));
      return;
    }

    if (page === 'create') {
      app.showPage(app.els.pageCreate);
      if (!state.categories.length) {
        app.loadCategories().then(function () {
          app.renderCreateCategories();
        });
      } else {
        app.renderCreateCategories();
      }
      return;
    }

    app.showErrorPage({
      code: 404,
      title: 'Страница пала в бою',
      message: 'Даже воины иногда теряют путь...'
    });
  };

  app.bindAuthEvents();
  app.bindPostsEvents();
  app.bindPMEvents();
  app.bindNotificationsEvents();
  if (app.els.appErrorHome) {
    app.els.appErrorHome.addEventListener('click', function () {
      app.navigate('/');
    });
  }
  if (app.els.appErrorBack) {
    app.els.appErrorBack.addEventListener('click', function () {
      if (window.history.length > 1) {
        window.history.back();
        return;
      }
      app.navigate('/');
    });
  }

  window.addEventListener('unhandledrejection', function (event) {
    var err = event && event.reason ? event.reason : {};
    var status = err && err.status;
    if (status === 404 || status === 405 || status === 500) {
      app.showErrorPage({
        code: status,
        title: status === 500 ? 'Сервер пал в Рагнарёке' : 'Страница пала в бою',
        message: status === 500 ? 'Кузнецы уже чинят щиты...' : 'Даже воины иногда теряют путь...'
      });
    }
  });

  window.addEventListener('popstate', app.route);
  app.checkAuth().then(function (ok) {
    if (!ok) app.route();
  });
  setInterval(function () {
    if (state.user) {
      app.loadConversations();
      app.loadNotifications();
    }
  }, 10000);
})();
