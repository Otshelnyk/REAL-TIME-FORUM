(function () {
  'use strict';

  var app = window.ForumApp = window.ForumApp || {};

  function getQueryParam(name) {
    try {
      var u = new URL(window.location.href);
      return u.searchParams.get(name);
    } catch (e) {
      return null;
    }
  }

  function parseCode(raw) {
    if (!raw) return null;
    var n = parseInt(String(raw), 10);
    if (isNaN(n)) return null;
    return n;
  }

  app.getErrorFromLocation = function () {
    var path = (typeof app.normalizePath === 'function')
      ? app.normalizePath(window.location.pathname)
      : (window.location.pathname || '/');

    if (path !== '/error') return null;

    var code = parseCode(getQueryParam('code')) || 500;
    var title = getQueryParam('title');
    var message = getQueryParam('message');

    return {
      code: code,
      title: title || undefined,
      message: message || undefined
    };
  };
})();

