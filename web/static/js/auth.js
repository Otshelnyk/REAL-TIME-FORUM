(function () {
  'use strict';

  var app = window.ForumApp;
  var state = app.state;
  var els = app.els;

  function setFieldState(el, fieldState) {
    if (!el) return;
    el.classList.remove('error', 'valid');
    if (fieldState === 'error') el.classList.add('error');
    if (fieldState === 'valid') el.classList.add('valid');
  }

  function clearRegisterBorders() {
    els.registerForm.querySelectorAll('input, select').forEach(function (el) {
      setFieldState(el, '');
    });
  }

  function isPasswordStrong(p) {
    if (!p || p.length < 6) return false;
    var hasLetter = /[a-zA-Z]/.test(p);
    var hasDigit = /\d/.test(p);
    return hasLetter && hasDigit;
  }

  function validateFirstName() {
    var el = els.registerForm.querySelector('[name=first_name]');
    var v = (el && el.value || '').trim();
    setFieldState(el, v.length >= 1 && v.length <= 50 ? 'valid' : (v.length > 50 ? 'error' : (el && el.dataset.touched ? 'error' : '')));
  }

  function validateLastName() {
    var el = els.registerForm.querySelector('[name=last_name]');
    var v = (el && el.value || '').trim();
    setFieldState(el, v.length >= 1 && v.length <= 50 ? 'valid' : (v.length > 50 ? 'error' : (el && el.dataset.touched ? 'error' : '')));
  }

  function validateNickname() {
    var el = els.registerForm.querySelector('[name=nickname]');
    var v = (el && el.value || '').trim();
    if (v.length < 2 || v.length > 30) {
      setFieldState(el, v.length > 30 || (el && el.dataset.touched) ? 'error' : '');
      return;
    }
    app.fetchJSON('/api/check-availability?nickname=' + encodeURIComponent(v)).then(function (data) {
      setFieldState(els.registerForm.querySelector('[name=nickname]'), data.nickname_available ? 'valid' : 'error');
    }).catch(function () { setFieldState(el, 'error'); });
  }

  function validateAge() {
    var el = els.registerForm.querySelector('[name=age]');
    var v = parseInt(el && el.value, 10);
    var ok = !isNaN(v) && v >= 1 && v <= 150;
    setFieldState(el, el && el.value !== '' ? (ok ? 'valid' : 'error') : (el && el.dataset.touched ? 'error' : ''));
  }

  function validateGender() {
    var el = els.registerForm.querySelector('[name=gender]');
    var v = (el && el.value || '').trim();
    setFieldState(el, v ? 'valid' : (el && el.dataset.touched ? 'error' : ''));
  }

  function validateEmail() {
    var el = els.registerForm.querySelector('[name=email]');
    var v = (el && el.value || '').trim();
    var emailOk = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v);
    if (!v || !emailOk) {
      setFieldState(el, (el && el.dataset.touched) || v.length > 0 ? 'error' : '');
      return;
    }
    app.fetchJSON('/api/check-availability?email=' + encodeURIComponent(v)).then(function (data) {
      setFieldState(els.registerForm.querySelector('[name=email]'), data.email_available ? 'valid' : 'error');
    }).catch(function () { setFieldState(el, 'error'); });
  }

  function validatePassword() {
    var el = els.registerForm.querySelector('[name=password]');
    var v = (el && el.value || '');
    setFieldState(el, v.length >= 6 ? (isPasswordStrong(v) ? 'valid' : 'error') : (v.length > 0 || (el && el.dataset.touched) ? 'error' : ''));
  }

  function validateConfirmPassword() {
    var passEl = els.registerForm.querySelector('[name=password]');
    var confirmEl = els.registerForm.querySelector('[name=confirm_password]');
    var pass = (passEl && passEl.value) || '';
    var confirm = (confirmEl && confirmEl.value) || '';
    if (!confirmEl) return;
    if (!confirm) {
      setFieldState(confirmEl, confirmEl.dataset.touched ? 'error' : '');
      return;
    }
    setFieldState(confirmEl, pass === confirm ? 'valid' : 'error');
  }

  var validateNicknameDebounced = app.debounce(validateNickname, 400);
  var validateEmailDebounced = app.debounce(validateEmail, 400);

  function switchAuthTab(tab) {
    if (tab === 'register') {
      app.showRegisterForm();
      app.navigate('/register');
      return;
    }
    app.showLoginForm();
    app.navigate('/login');
  }

  app.showLoginForm = function () {
    document.querySelectorAll('.auth-tab').forEach(function (t) {
      t.classList.toggle('active', t.dataset.tab === 'login');
    });
    els.loginForm.classList.remove('hidden');
    els.registerForm.classList.add('hidden');
    app.hide(els.authError);
  };

  app.showRegisterForm = function () {
    document.querySelectorAll('.auth-tab').forEach(function (t) {
      t.classList.toggle('active', t.dataset.tab === 'register');
    });
    els.registerForm.classList.remove('hidden');
    els.loginForm.classList.add('hidden');
    app.hide(els.authError);
    clearRegisterBorders();
  };

  app.checkAuth = function () {
    return app.fetchJSON('/api/me').then(function (data) {
      if (data.success && data.data) {
        state.user = data.data;
        app.hide(els.authScreen);
        app.show(els.appShell);
        els.headerUsername.textContent = state.user.nickname;
        app.renderHeaderAvatar();
        app.loadCategories();
        app.loadFeed();
        app.loadConversations();
        if (typeof app.loadNotifications === 'function') app.loadNotifications();
        if (typeof app.startNotificationsPolling === 'function') app.startNotificationsPolling();
        app.connectWS();
        app.route();
        return true;
      }
      throw new Error('Not authenticated');
    }).catch(function () {
      state.user = null;
      app.show(els.authScreen);
      app.hide(els.appShell);
        app.renderHeaderAvatar();
        if (typeof app.resetNotifications === 'function') app.resetNotifications();
        if (typeof app.stopNotificationsPolling === 'function') app.stopNotificationsPolling();
      if (state.ws) { state.ws.close(); state.ws = null; }
      app.route();
      return false;
    });
  };

  app.bindAuthEvents = function () {
    if (els.headerAvatarImg) {
      els.headerAvatarImg.addEventListener('error', function () {
        if (state.user) state.user.avatar_url = '';
        app.renderHeaderAvatar();
      });
    }

    els.registerForm.querySelectorAll('[name=first_name]').forEach(function (el) {
      el.addEventListener('input', validateFirstName);
      el.addEventListener('blur', function () { el.dataset.touched = '1'; validateFirstName(); });
    });
    els.registerForm.querySelectorAll('[name=last_name]').forEach(function (el) {
      el.addEventListener('input', validateLastName);
      el.addEventListener('blur', function () { el.dataset.touched = '1'; validateLastName(); });
    });
    els.registerForm.querySelectorAll('[name=nickname]').forEach(function (el) {
      el.addEventListener('input', function () { validateNicknameDebounced(); });
      el.addEventListener('blur', function () { el.dataset.touched = '1'; validateNicknameDebounced(); });
    });
    els.registerForm.querySelectorAll('[name=age]').forEach(function (el) {
      el.addEventListener('input', validateAge);
      el.addEventListener('blur', function () { el.dataset.touched = '1'; validateAge(); });
    });
    els.registerForm.querySelectorAll('[name=gender]').forEach(function (el) {
      el.addEventListener('change', validateGender);
      el.addEventListener('blur', function () { el.dataset.touched = '1'; validateGender(); });
    });
    els.registerForm.querySelectorAll('[name=email]').forEach(function (el) {
      el.addEventListener('input', function () { validateEmailDebounced(); });
      el.addEventListener('blur', function () { el.dataset.touched = '1'; validateEmailDebounced(); });
    });
    els.registerForm.querySelectorAll('[name=password]').forEach(function (el) {
      el.addEventListener('input', validatePassword);
      el.addEventListener('blur', function () { el.dataset.touched = '1'; validatePassword(); });
    });
    els.registerForm.querySelectorAll('[name=confirm_password]').forEach(function (el) {
      el.addEventListener('input', validateConfirmPassword);
      el.addEventListener('blur', function () { el.dataset.touched = '1'; validateConfirmPassword(); });
    });

    document.querySelectorAll('.auth-inline-switch').forEach(function (btn) {
      btn.addEventListener('click', function () {
        switchAuthTab(btn.dataset.tabSwitch);
      });
    });

    document.querySelectorAll('.auth-tab').forEach(function (btn) {
      btn.addEventListener('click', function () {
        switchAuthTab(this.dataset.tab);
      });
    });

    els.loginForm.addEventListener('submit', function (e) {
      e.preventDefault();
      var login = this.login.value.trim();
      var password = this.password.value;
      app.hide(els.authError);
      app.fetchJSON('/api/login', {
        method: 'POST',
        json: true,
        body: JSON.stringify({ login: login, password: password })
      }).then(function (data) {
        if (data.success) app.checkAuth();
        else { els.authError.textContent = data.message || 'Не удалось войти в аккаунт'; app.show(els.authError); }
      }).catch(function (err) {
        els.authError.textContent = err.message || 'Не удалось войти в аккаунт';
        app.show(els.authError);
      });
    });

    els.registerForm.addEventListener('submit', function (e) {
      e.preventDefault();
      app.hide(els.authError);
      els.registerForm.querySelectorAll('input, select').forEach(function (el) { el.dataset.touched = '1'; });
      validateFirstName();
      validateLastName();
      validateAge();
      validateGender();
      validatePassword();
      validateConfirmPassword();
      var fd = new FormData(els.registerForm);
      var body = {
        nickname: (fd.get('nickname') || '').trim(),
        age: parseInt(fd.get('age'), 10) || 0,
        gender: (fd.get('gender') || '').trim(),
        first_name: (fd.get('first_name') || '').trim(),
        last_name: (fd.get('last_name') || '').trim(),
        email: (fd.get('email') || '').trim(),
        password: fd.get('password') || ''
      };
      var nicknameEl = els.registerForm.querySelector('[name=nickname]');
      var emailEl = els.registerForm.querySelector('[name=email]');
      if (body.nickname.length < 2 || body.nickname.length > 30) setFieldState(nicknameEl, 'error');
      var emailFormatOk = body.email && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(body.email);
      if (!emailFormatOk) setFieldState(emailEl, 'error');
      var hasError = els.registerForm.querySelectorAll('.error').length > 0;
      if (hasError) {
        els.authError.style.color = '';
        els.authError.textContent = 'Проверьте поля, выделенные красным.';
        app.show(els.authError);
        return;
      }
      if (body.password !== ((fd.get('confirm_password') || ''))) {
        els.authError.style.color = '';
        els.authError.textContent = 'Пароли не совпадают.';
        app.show(els.authError);
        setFieldState(els.registerForm.querySelector('[name=confirm_password]'), 'error');
        return;
      }
      if (body.age < 1 || body.age > 150) {
        els.authError.textContent = 'Возраст должен быть от 1 до 150.';
        app.show(els.authError);
        return;
      }
      fetch('/api/check-availability?nickname=' + encodeURIComponent(body.nickname) + '&email=' + encodeURIComponent(body.email), { credentials: 'same-origin' })
        .then(function (res) { return res.json(); })
        .then(function (avail) {
          var nickTaken = avail.nickname_available === false;
          var emailTaken = avail.email_available === false;
          if (nickTaken) setFieldState(nicknameEl, 'error');
          else setFieldState(nicknameEl, 'valid');
          if (emailTaken) setFieldState(emailEl, 'error');
          else setFieldState(emailEl, 'valid');
          if (nickTaken || emailTaken) {
            els.authError.style.color = '';
            if (nickTaken && emailTaken) els.authError.textContent = 'Никнейм и email уже используются.';
            else if (nickTaken) els.authError.textContent = 'Этот никнейм уже занят.';
            else els.authError.textContent = 'Этот email уже используется.';
            app.show(els.authError);
            return null;
          }
          return app.fetchJSON('/api/register', {
            method: 'POST',
            json: true,
            body: JSON.stringify(body)
          });
        })
        .then(function (data) {
          if (!data) return;
          if (data.success) {
            els.authError.textContent = 'Регистрация прошла успешно. Теперь войдите.';
            els.authError.style.color = 'green';
            app.show(els.authError);
            app.showLoginForm();
            app.navigate('/login', true);
          } else {
            els.authError.style.color = '';
            els.authError.textContent = data.message || 'Не удалось зарегистрироваться';
            app.show(els.authError);
            var msg = (data.message || '').toLowerCase();
            if (msg.indexOf('никнейм') !== -1) setFieldState(nicknameEl, 'error');
            if (msg.indexOf('email') !== -1 || msg.indexOf('почт') !== -1) setFieldState(emailEl, 'error');
          }
        })
        .catch(function (err) {
          els.authError.style.color = '';
          els.authError.textContent = err.message || 'Не удалось зарегистрироваться';
          app.show(els.authError);
        });
    });

    els.btnLogout.addEventListener('click', function () {
      app.fetchJSON('/api/logout', { method: 'POST' }).then(function () {
        state.user = null;
        app.hide(els.appShell);
        app.show(els.authScreen);
        app.renderHeaderAvatar();
        if (typeof app.resetNotifications === 'function') app.resetNotifications();
        if (typeof app.stopNotificationsPolling === 'function') app.stopNotificationsPolling();
        if (state.ws) { state.ws.close(); state.ws = null; }
        app.showLoginForm();
        app.navigate('/login');
      });
    });

    els.avatarChangeBtn.addEventListener('click', function () {
      if (els.avatarInput) els.avatarInput.click();
    });

    els.avatarInput.addEventListener('change', function () {
      if (!this.files || !this.files.length) return;
      var file = this.files[0];
      var fd = new FormData();
      fd.append('avatar', file);
      fetch('/api/profile/avatar', {
        method: 'POST',
        body: fd,
        credentials: 'same-origin'
      }).then(function (res) {
        return res.json();
      }).then(function (data) {
        if (!data.success || !data.data || !data.data.avatar_url) {
          alert(data.message || 'Failed to update avatar');
          return;
        }
        state.user.avatar_url = data.data.avatar_url;
        app.renderHeaderAvatar();
        app.route();
      }).catch(function () {
        alert('Failed to update avatar');
      }).finally(function () {
        els.avatarInput.value = '';
      });
    });
  };
})();
