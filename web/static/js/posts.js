(function () {
  'use strict';

  var app = window.ForumApp;
  var state = app.state;
  var els = app.els;
  var draftCategories = null;

  function sameCategorySet(a, b) {
    var x = (a || []).slice().sort(function (m, n) { return m - n; });
    var y = (b || []).slice().sort(function (m, n) { return m - n; });
    if (x.length !== y.length) return false;
    for (var i = 0; i < x.length; i++) {
      if (x[i] !== y[i]) return false;
    }
    return true;
  }

  app.loadCategories = function () {
    return app.fetchJSON('/api/categories').then(function (data) {
      if (data.success && data.data) state.categories = data.data;
      return state.categories;
    });
  };

  function bindFeedNavigation() {
    els.feedPosts.querySelectorAll('.post-link').forEach(function (a) {
      a.addEventListener('click', function (e) {
        e.preventDefault();
        var id = parseInt(a.dataset.postId, 10);
        if (!isNaN(id)) app.navigate('/post/' + id);
      });
    });
    els.feedPosts.querySelectorAll('.feed-link').forEach(function (a) {
      a.addEventListener('click', function (e) {
        e.preventDefault();
        app.navigate('/');
      });
    });
  }

  function bindFeedLikeButtons() {
    els.feedPosts.querySelectorAll('.like-post-btn, .dislike-post-btn').forEach(function (btn) {
      btn.addEventListener('click', function () {
        var postId = this.dataset.postId;
        var like = this.dataset.like;
        var fd = new FormData();
        fd.append('post_id', postId);
        fd.append('like', like);
        fetch('/api/like_post', { method: 'POST', body: fd, credentials: 'same-origin' })
          .then(function (r) { return r.json(); })
          .then(function (data) {
            if (data.success && data.data) {
              var card = btn.closest('.blog-post-card');
              var likeBtn = card.querySelector('.like-post-btn');
              var dislikeBtn = card.querySelector('.dislike-post-btn');
              if (likeBtn) likeBtn.querySelector('.count').textContent = String(data.data.likes || 0);
              if (dislikeBtn) dislikeBtn.querySelector('.count').textContent = String(data.data.dislikes || 0);
              if (likeBtn && dislikeBtn) {
                likeBtn.classList.toggle('active', !!data.data.is_liked);
                dislikeBtn.classList.toggle('active', !!data.data.is_disliked);
              }
            }
          });
      });
    });
  }

  function bindCommentLikeButtons(root) {
    if (!root) return;
    root.querySelectorAll('.like-comment-btn, .dislike-comment-btn').forEach(function (btn) {
      btn.addEventListener('click', function () {
        var commentId = this.dataset.commentId;
        var like = this.dataset.like;
        var fd = new FormData();
        fd.append('comment_id', commentId);
        fd.append('like', like);
        fetch('/api/like_comment', { method: 'POST', body: fd, credentials: 'same-origin' })
          .then(function (r) { return r.json(); })
          .then(function (data) {
            if (!data.success || !data.data) return;
            var item = btn.closest('.comment-item');
            if (!item) return;
            var likeBtn = item.querySelector('.like-comment-btn');
            var dislikeBtn = item.querySelector('.dislike-comment-btn');
            if (likeBtn) likeBtn.querySelector('.count').textContent = String(data.data.likes || 0);
            if (dislikeBtn) dislikeBtn.querySelector('.count').textContent = String(data.data.dislikes || 0);
            if (likeBtn && dislikeBtn) {
              likeBtn.classList.toggle('active', !!data.data.is_liked);
              dislikeBtn.classList.toggle('active', !!data.data.is_disliked);
            }
          });
      });
    });
  }

  function renderCategoryFilter() {
    draftCategories = (state.currentCategories || []).slice();
    var selected = draftCategories;
    var selectedSet = {};
    selected.forEach(function (id) { selectedSet[String(id)] = true; });
    var selectedCount = selected.length;
    var summary = selectedCount ? ('Categories: ' + selectedCount) : 'Categories';
    var html = '<div class="category-cell">' +
      '<button type="button" class="category-cell-toggle">' + app.escapeHtml(summary) + '</button>' +
      '<div class="category-cell-menu hidden">';
    (state.categories || []).forEach(function (c) {
      var cid = String(app.categoryId(c));
      var checked = selectedSet[cid] ? ' checked' : '';
      html += '<label class="category-cell-item"><input type="checkbox" data-category-id="' + cid + '"' + checked + '> ' + app.escapeHtml(app.categoryName(c)) + '</label>';
    });
    html += '</div></div>';
    html += '<button type="button" class="category-apply-btn">Apply</button>';
    html += '<button type="button" class="category-action-btn ' + (!selectedCount && !state.currentFilter ? 'active' : '') + '" data-filter="">All</button>';
    if (state.user) {
      html += '<button type="button" class="category-action-btn ' + (state.currentFilter === 'myposts' ? 'active' : '') + '" data-filter="myposts">My Posts</button>';
      html += '<button type="button" class="category-action-btn ' + (state.currentFilter === 'liked' ? 'active' : '') + '" data-filter="liked">Liked</button>';
    }
    els.categoryFilter.innerHTML = html;

    var toggle = els.categoryFilter.querySelector('.category-cell-toggle');
    var menu = els.categoryFilter.querySelector('.category-cell-menu');
    var applyBtn = els.categoryFilter.querySelector('.category-apply-btn');
    function refreshSummary() {
      if (!toggle) return;
      var count = (draftCategories || []).length;
      toggle.textContent = count ? ('Categories: ' + count) : 'Categories';
    }
    function refreshApplyState() {
      if (!applyBtn) return;
      applyBtn.disabled = sameCategorySet(draftCategories, state.currentCategories || []);
    }
    refreshSummary();
    refreshApplyState();

    if (toggle && menu) {
      toggle.addEventListener('click', function () {
        menu.classList.toggle('hidden');
      });
      menu.querySelectorAll('input[type="checkbox"]').forEach(function (cb) {
        cb.addEventListener('change', function () {
          var id = parseInt(cb.dataset.categoryId, 10);
          if (isNaN(id)) return;
          var next = (draftCategories || []).slice();
          var idx = next.indexOf(id);
          if (cb.checked && idx === -1) next.push(id);
          if (!cb.checked && idx !== -1) next.splice(idx, 1);
          draftCategories = next;
          refreshSummary();
          refreshApplyState();
        });
      });
    }
    if (applyBtn) {
      applyBtn.addEventListener('click', function () {
        state.currentCategories = (draftCategories || []).slice();
        state.currentCategory = '';
        state.currentFilter = '';
        state.currentFeedPage = 1;
        if (menu) menu.classList.add('hidden');
        app.navigate('/');
      });
    }

    els.categoryFilter.querySelectorAll('.category-action-btn').forEach(function (btn) {
      btn.addEventListener('click', function () {
        var filter = btn.dataset.filter || '';
        state.currentFilter = filter;
        draftCategories = [];
        if (filter) {
          state.currentCategories = [];
          state.currentCategory = '';
        } else {
          state.currentCategories = [];
          state.currentCategory = '';
        }
        state.currentFeedPage = 1;
        app.navigate('/');
      });
    });
  }

  function renderFeedPagination(page, totalPages) {
    if (!totalPages || totalPages <= 1) {
      els.feedPagination.innerHTML = '';
      return;
    }
    var html = '';
    if (page > 1) html += '<a href="/feed/' + (page - 1) + '">Previous</a>';
    html += ' <span>Page ' + page + ' of ' + totalPages + '</span> ';
    if (page < totalPages) html += '<a href="/feed/' + (page + 1) + '">Next</a>';
    els.feedPagination.innerHTML = html;
    els.feedPagination.querySelectorAll('a').forEach(function (a) {
      a.addEventListener('click', function (e) {
        e.preventDefault();
        var m = a.getAttribute('href').match(/feed\/(\d+)/);
        if (m) {
          state.currentFeedPage = parseInt(m[1], 10);
          app.navigate('/feed/' + state.currentFeedPage);
        }
      });
    });
  }

  app.loadFeed = function () {
    var q = '?page=' + state.currentFeedPage;
    if (state.currentCategories && state.currentCategories.length) {
      q += '&categories=' + encodeURIComponent(state.currentCategories.join(','));
    } else if (state.currentCategory) {
      q += '&category=' + encodeURIComponent(state.currentCategory);
    }
    if (state.currentFilter) q += '&filter=' + encodeURIComponent(state.currentFilter);
    app.fetchJSON('/api/posts' + q).then(function (data) {
      if (!data.success) return;
      var d = data.data;
      renderCategoryFilter();
      els.feedPosts.innerHTML = (d.posts || []).map(function (p) {
        var cats = (p.categories || []).map(function (c) {
          return '<a href="/" class="badge feed-link">' + app.escapeHtml(app.categoryName(c)) + '</a>';
        }).join(' ');
        return (
          '<article class="blog-post-card" data-post-id="' + p.id + '">' +
          '<div class="post-meta-row">' + app.avatarHTML(p.author_avatar, p.author, 'avatar-sm') + '<p class="blog-post-meta">' + app.escapeHtml(p.created_at) + ' by ' + app.escapeHtml(p.author) + ' ' + cats + ' | Comments: ' + (p.comment_count || 0) + '</p></div>' +
          '<h2><a href="/post/' + p.id + '" class="post-link" data-post-id="' + p.id + '">' + app.escapeHtml(p.title) + '</a></h2>' +
          '<div class="post-content">' + app.escapeHtml((p.content || '').slice(0, 300)) + ((p.content && p.content.length > 300) ? '...' : '') + '</div>' +
          '<div class="post-actions yt-reactions">' +
          '<button type="button" class="like-post-btn yt-reaction-btn" data-post-id="' + p.id + '" data-like="1"><span class="icon">👍</span><span class="count">' + (p.likes || 0) + '</span></button>' +
          '<button type="button" class="dislike-post-btn yt-reaction-btn" data-post-id="' + p.id + '" data-like="0"><span class="icon">👎</span><span class="count">' + (p.dislikes || 0) + '</span></button>' +
          '</div></article>'
        );
      }).join('') || '<p>No posts yet.</p>';

      bindFeedLikeButtons();
      bindFeedNavigation();
      renderFeedPagination(d.page, d.total_pages);
    });
  };

  app.loadPostDetail = function (postId) {
    app.fetchJSON('/api/post/' + postId).then(function (data) {
      if (!data.success || !data.data) return;
      var p = data.data.post;
      var comments = data.data.comments || [];
      els.postDetailContent.innerHTML = (
        '<article class="blog-post-card">' +
        '<div class="post-meta-row">' + app.avatarHTML(p.author_avatar, p.author, 'avatar-md') + '<p class="blog-post-meta">' + app.escapeHtml(p.created_at) + ' by ' + app.escapeHtml(p.author) + '</p></div>' +
        '<h2>' + app.escapeHtml(p.title) + '</h2>' +
        '<div class="post-content">' + app.escapeHtml(p.content || '') + '</div>' +
        '<div class="post-actions yt-reactions">' +
        '<button type="button" class="like-post-btn yt-reaction-btn" data-post-id="' + p.id + '" data-like="1"><span class="icon">👍</span><span class="count">' + (p.likes || 0) + '</span></button>' +
        '<button type="button" class="dislike-post-btn yt-reaction-btn" data-post-id="' + p.id + '" data-like="0"><span class="icon">👎</span><span class="count">' + (p.dislikes || 0) + '</span></button>' +
        '</div></article>'
      );
      els.postDetailComments.innerHTML = comments.map(function (c) {
        return (
          '<div class="comment-item" data-comment-id="' + c.id + '">' +
          '<div class="comment-row">' + app.avatarHTML(c.author_avatar, c.author, 'avatar-xs') + '<div><div class="comment-meta">' + app.escapeHtml(c.author) + ' • ' + app.escapeHtml(c.created_at) + '</div>' +
          '<div class="comment-content">' + app.escapeHtml(c.content) + '</div>' +
          '<div class="comment-actions-inline yt-reactions">' +
          '<button type="button" class="like-comment-btn yt-reaction-btn" data-comment-id="' + c.id + '" data-like="1"><span class="icon">👍</span><span class="count">' + (c.likes || 0) + '</span></button>' +
          '<button type="button" class="dislike-comment-btn yt-reaction-btn" data-comment-id="' + c.id + '" data-like="0"><span class="icon">👎</span><span class="count">' + (c.dislikes || 0) + '</span></button>' +
          '</div></div></div></div>'
        );
      }).join('');
      bindCommentLikeButtons(els.postDetailComments);
      els.commentForm.classList.remove('hidden');
      els.commentForm.dataset.postId = postId;
      els.commentForm.comment.value = '';
      els.postDetailContent.querySelectorAll('.like-post-btn, .dislike-post-btn').forEach(function (btn) {
        btn.addEventListener('click', function () {
          var fd = new FormData();
          fd.append('post_id', postId);
          fd.append('like', this.dataset.like);
          fetch('/api/like_post', { method: 'POST', body: fd, credentials: 'same-origin' })
            .then(function (r) { return r.json(); })
            .then(function (d) {
              if (d.success && d.data) {
                var actions = btn.closest('.post-actions');
                var likeBtn = actions.querySelector('.like-post-btn');
                var dislikeBtn = actions.querySelector('.dislike-post-btn');
                if (likeBtn) likeBtn.querySelector('.count').textContent = String(d.data.likes || 0);
                if (dislikeBtn) dislikeBtn.querySelector('.count').textContent = String(d.data.dislikes || 0);
                if (likeBtn && dislikeBtn) {
                  likeBtn.classList.toggle('active', !!d.data.is_liked);
                  dislikeBtn.classList.toggle('active', !!d.data.is_disliked);
                }
              }
            });
        });
      });
    });
  };

  app.renderCreateCategories = function () {
    els.createPostCategories.innerHTML = (state.categories || []).map(function (c) {
      return '<label><input type="checkbox" name="category_ids" value="' + app.categoryId(c) + '"> ' + app.escapeHtml(app.categoryName(c)) + '</label>';
    }).join('');
  };

  app.bindPostsEvents = function () {
    els.commentForm.addEventListener('submit', function (e) {
      e.preventDefault();
      var postId = this.dataset.postId;
      var content = this.comment.value.trim();
      if (!content) return;
      var fd = new FormData();
      fd.append('post_id', postId);
      fd.append('comment', content);
      fetch('/api/comment', { method: 'POST', body: fd, credentials: 'same-origin' })
        .then(function (r) { return r.json(); })
        .then(function (data) {
          if (data.success && data.data) {
            var c = data.data;
            var div = document.createElement('div');
            div.className = 'comment-item';
            div.dataset.commentId = c.id;
            div.innerHTML = '<div class="comment-row">' + app.avatarHTML(c.author_avatar, c.author, 'avatar-xs') + '<div><div class="comment-meta">' + app.escapeHtml(c.author) + ' • ' + app.escapeHtml(c.created_at) + '</div><div class="comment-content">' + app.escapeHtml(c.content) + '</div><div class="comment-actions-inline yt-reactions"><button type="button" class="like-comment-btn yt-reaction-btn" data-comment-id="' + c.id + '" data-like="1"><span class="icon">👍</span><span class="count">' + (c.likes || 0) + '</span></button><button type="button" class="dislike-comment-btn yt-reaction-btn" data-comment-id="' + c.id + '" data-like="0"><span class="icon">👎</span><span class="count">' + (c.dislikes || 0) + '</span></button></div></div></div>';
            els.postDetailComments.appendChild(div);
            bindCommentLikeButtons(div);
            els.commentForm.comment.value = '';
          } else alert(data.message || 'Failed to post comment');
        });
    });

    els.createPostForm.addEventListener('submit', function (e) {
      e.preventDefault();
      var title = this.title.value.trim();
      var content = this.content.value.trim();
      var ids = [];
      els.createPostForm.querySelectorAll('input[name="category_ids"]:checked').forEach(function (cb) {
        ids.push(parseInt(cb.value, 10));
      });
      if (!title || !content || ids.length === 0) {
        alert('Title, content and at least one category required');
        return;
      }
      app.fetchJSON('/api/post', {
        method: 'POST',
        json: true,
        body: JSON.stringify({ title: title, content: content, category_ids: ids })
      }).then(function (data) {
        if (data.success && data.data && data.data.id) {
          app.navigate('/post/' + data.data.id);
        } else alert(data.message || 'Failed to create post');
      }).catch(function (err) { alert(err.message || 'Failed'); });
    });
  };
})();
