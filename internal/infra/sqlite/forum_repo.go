package sqlite

import (
	"database/sql"
	"strings"
	"time"

	"github.com/ndanbaev/forum/internal/app/ports"
)

type ForumRepo struct {
	db *sql.DB
}

func NewForumRepo(db *sql.DB) *ForumRepo {
	return &ForumRepo{db: db}
}

func (r *ForumRepo) ListCategories() ([]ports.Category, error) {
	rows, err := r.db.Query("SELECT id, name FROM categories ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ports.Category
	for rows.Next() {
		var c ports.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			continue
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *ForumRepo) ListPosts(p ports.ForumListPostsParams) (ports.ForumListPostsResult, error) {
	page := p.Page
	if page <= 0 {
		page = 1
	}
	pageSize := p.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	var totalPosts int
	var rows *sql.Rows
	var err error

	if p.Filter == "myposts" && p.UserID > 0 {
		_ = r.db.QueryRow("SELECT COUNT(*) FROM posts WHERE user_id = ?", p.UserID).Scan(&totalPosts)
		rows, err = r.db.Query(`SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.nickname, u.avatar_url FROM posts p JOIN users u ON p.user_id = u.id WHERE p.user_id = ? ORDER BY p.created_at DESC LIMIT ? OFFSET ?`, p.UserID, pageSize, offset)
	} else if p.Filter == "liked" && p.UserID > 0 {
		_ = r.db.QueryRow("SELECT COUNT(DISTINCT p.id) FROM posts p JOIN post_likes pl ON p.id = pl.post_id WHERE pl.user_id = ? AND pl.is_like = 1", p.UserID).Scan(&totalPosts)
		rows, err = r.db.Query(`SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.nickname, u.avatar_url FROM posts p JOIN users u ON p.user_id = u.id JOIN post_likes pl ON p.id = pl.post_id WHERE pl.user_id = ? AND pl.is_like = 1 GROUP BY p.id ORDER BY p.created_at DESC LIMIT ? OFFSET ?`, p.UserID, pageSize, offset)
	} else if len(p.SelectedCategories) > 0 {
		placeholders := make([]string, len(p.SelectedCategories))
		countArgs := make([]any, len(p.SelectedCategories))
		queryArgs := make([]any, 0, len(p.SelectedCategories)+2)
		for i, cid := range p.SelectedCategories {
			placeholders[i] = "?"
			countArgs[i] = cid
			queryArgs = append(queryArgs, cid)
		}
		inClause := strings.Join(placeholders, ",")
		countSQL := "SELECT COUNT(DISTINCT p.id) FROM posts p JOIN post_categories pc ON p.id = pc.post_id WHERE pc.category_id IN (" + inClause + ")"
		_ = r.db.QueryRow(countSQL, countArgs...).Scan(&totalPosts)
		querySQL := "SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.nickname, u.avatar_url FROM posts p JOIN users u ON p.user_id = u.id JOIN post_categories pc ON p.id = pc.post_id WHERE pc.category_id IN (" + inClause + ") GROUP BY p.id ORDER BY p.created_at DESC LIMIT ? OFFSET ?"
		queryArgs = append(queryArgs, pageSize, offset)
		rows, err = r.db.Query(querySQL, queryArgs...)
	} else {
		_ = r.db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&totalPosts)
		rows, err = r.db.Query(`SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.nickname, u.avatar_url FROM posts p JOIN users u ON p.user_id = u.id ORDER BY p.created_at DESC LIMIT ? OFFSET ?`, pageSize, offset)
	}
	if err != nil {
		return ports.ForumListPostsResult{}, err
	}
	defer rows.Close()

	var posts []ports.ForumPostFeedItemRow
	for rows.Next() {
		var it ports.ForumPostFeedItemRow
		var createdAt string
		if err := rows.Scan(&it.ID, &it.UserID, &it.Title, &it.Content, &createdAt, &it.Author, &it.AuthorAvatar); err != nil {
			continue
		}
		it.CreatedAtRaw = createdAt
		_ = r.db.QueryRow("SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND is_like = 1", it.ID).Scan(&it.Likes)
		_ = r.db.QueryRow("SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND is_like = 0", it.ID).Scan(&it.Dislikes)
		_ = r.db.QueryRow("SELECT COUNT(*) FROM comments WHERE post_id = ?", it.ID).Scan(&it.CommentCount)
		catRows, _ := r.db.Query("SELECT c.id, c.name FROM categories c JOIN post_categories pc ON c.id = pc.category_id WHERE pc.post_id = ?", it.ID)
		var cats []ports.Category
		for catRows.Next() {
			var c ports.Category
			_ = catRows.Scan(&c.ID, &c.Name)
			cats = append(cats, c)
		}
		catRows.Close()
		it.Categories = cats
		posts = append(posts, it)
	}

	totalPages := (totalPosts + pageSize - 1) / pageSize
	return ports.ForumListPostsResult{Posts: posts, Page: page, TotalPages: totalPages, TotalPosts: totalPosts}, nil
}

func (r *ForumRepo) GetPost(postID int) (ports.ForumGetPostResult, error) {
	var p ports.ForumPostDetailRow
	var createdAt string
	err := r.db.QueryRow(`SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.nickname, u.avatar_url FROM posts p JOIN users u ON p.user_id = u.id WHERE p.id = ?`, postID).
		Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &createdAt, &p.Author, &p.AuthorAvatar)
	if err != nil {
		return ports.ForumGetPostResult{}, err
	}
	p.CreatedAtRaw = createdAt
	_ = r.db.QueryRow("SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND is_like = 1", p.ID).Scan(&p.Likes)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND is_like = 0", p.ID).Scan(&p.Dislikes)
	catRows, _ := r.db.Query("SELECT c.id, c.name FROM categories c JOIN post_categories pc ON c.id = pc.category_id WHERE pc.post_id = ?", p.ID)
	for catRows.Next() {
		var c ports.Category
		_ = catRows.Scan(&c.ID, &c.Name)
		p.Categories = append(p.Categories, c)
	}
	catRows.Close()

	commentRows, _ := r.db.Query(`SELECT c.id, c.post_id, c.user_id, c.content, c.created_at, u.nickname, u.avatar_url FROM comments c JOIN users u ON c.user_id = u.id WHERE c.post_id = ? ORDER BY c.created_at ASC`, p.ID)
	defer commentRows.Close()
	var comments []ports.ForumCommentDetailRow
	for commentRows.Next() {
		var c ports.ForumCommentDetailRow
		var cAt string
		_ = commentRows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &cAt, &c.Author, &c.AuthorAvatar)
		c.CreatedAtRaw = cAt
		_ = r.db.QueryRow("SELECT COUNT(*) FROM comment_likes WHERE comment_id = ? AND is_like = 1", c.ID).Scan(&c.Likes)
		_ = r.db.QueryRow("SELECT COUNT(*) FROM comment_likes WHERE comment_id = ? AND is_like = 0", c.ID).Scan(&c.Dislikes)
		comments = append(comments, c)
	}

	return ports.ForumGetPostResult{Post: p, Comments: comments}, nil
}

func (r *ForumRepo) CreatePost(userID int, title, content string, categoryIDs []int, createdAt time.Time) (int64, error) {
	res, err := r.db.Exec("INSERT INTO posts (user_id, title, content, created_at) VALUES (?, ?, ?, ?)", userID, title, content, createdAt)
	if err != nil {
		return 0, err
	}
	postID, _ := res.LastInsertId()
	for _, cid := range categoryIDs {
		if cid > 0 {
			_, _ = r.db.Exec("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)", postID, cid)
		}
	}
	return postID, nil
}

func (r *ForumRepo) AddComment(userID int, postID int, content string, createdAt time.Time) (int64, error) {
	res, err := r.db.Exec("INSERT INTO comments (post_id, user_id, content, created_at) VALUES (?, ?, ?, ?)", postID, userID, content, createdAt)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (r *ForumRepo) GetCommentByID(commentID int64) (ports.ForumCommentDetailRow, error) {
	var out ports.ForumCommentDetailRow
	var createdAt string
	err := r.db.QueryRow(`SELECT comments.id, comments.post_id, comments.user_id, comments.content, comments.created_at, users.nickname, users.avatar_url
		FROM comments JOIN users ON comments.user_id = users.id WHERE comments.id = ?`, commentID).
		Scan(&out.ID, &out.PostID, &out.UserID, &out.Content, &createdAt, &out.Author, &out.AuthorAvatar)
	if err != nil {
		return out, err
	}
	out.CreatedAtRaw = createdAt
	_ = r.db.QueryRow("SELECT COUNT(*) FROM comment_likes WHERE comment_id = ? AND is_like = 1", out.ID).Scan(&out.Likes)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM comment_likes WHERE comment_id = ? AND is_like = 0", out.ID).Scan(&out.Dislikes)
	return out, nil
}

func (r *ForumRepo) GetPostOwnerAndTitle(postID int) (ownerID int, title string, err error) {
	err = r.db.QueryRow("SELECT user_id, title FROM posts WHERE id = ?", postID).Scan(&ownerID, &title)
	return ownerID, title, err
}

func (r *ForumRepo) TogglePostReaction(userID int, postID int, isLike bool) (ports.ForumReactionResult, error) {
	hasReaction := false
	currentIsLike := false

	var existingLike bool
	err := r.db.QueryRow("SELECT is_like FROM post_likes WHERE post_id = ? AND user_id = ?", postID, userID).Scan(&existingLike)
	switch err {
	case sql.ErrNoRows:
		_, err = r.db.Exec("INSERT INTO post_likes (post_id, user_id, is_like) VALUES (?, ?, ?)", postID, userID, isLike)
		if err != nil {
			return ports.ForumReactionResult{}, err
		}
		hasReaction = true
		currentIsLike = isLike
	case nil:
		if existingLike == isLike {
			_, err = r.db.Exec("DELETE FROM post_likes WHERE post_id = ? AND user_id = ?", postID, userID)
			if err != nil {
				return ports.ForumReactionResult{}, err
			}
			hasReaction = false
		} else {
			_, err = r.db.Exec("UPDATE post_likes SET is_like = ? WHERE post_id = ? AND user_id = ?", isLike, postID, userID)
			if err != nil {
				return ports.ForumReactionResult{}, err
			}
			hasReaction = true
			currentIsLike = isLike
		}
	default:
		return ports.ForumReactionResult{}, err
	}

	var out ports.ForumReactionResult
	_ = r.db.QueryRow("SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND is_like = 1", postID).Scan(&out.Likes)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND is_like = 0", postID).Scan(&out.Dislikes)
	out.HasReaction = hasReaction
	out.CurrentIsLike = currentIsLike
	return out, nil
}

func (r *ForumRepo) ToggleCommentReaction(userID int, commentID int, isLike bool) (ports.ForumReactionResult, int, error) {
	hasReaction := false
	currentIsLike := false

	var existingLike bool
	err := r.db.QueryRow("SELECT is_like FROM comment_likes WHERE comment_id = ? AND user_id = ?", commentID, userID).Scan(&existingLike)
	switch err {
	case sql.ErrNoRows:
		_, err = r.db.Exec("INSERT INTO comment_likes (comment_id, user_id, is_like) VALUES (?, ?, ?)", commentID, userID, isLike)
		if err != nil {
			return ports.ForumReactionResult{}, 0, err
		}
		hasReaction = true
		currentIsLike = isLike
	case nil:
		if existingLike == isLike {
			_, err = r.db.Exec("DELETE FROM comment_likes WHERE comment_id = ? AND user_id = ?", commentID, userID)
			if err != nil {
				return ports.ForumReactionResult{}, 0, err
			}
			hasReaction = false
		} else {
			_, err = r.db.Exec("UPDATE comment_likes SET is_like = ? WHERE comment_id = ? AND user_id = ?", isLike, commentID, userID)
			if err != nil {
				return ports.ForumReactionResult{}, 0, err
			}
			hasReaction = true
			currentIsLike = isLike
		}
	default:
		return ports.ForumReactionResult{}, 0, err
	}

	var out ports.ForumReactionResult
	_ = r.db.QueryRow("SELECT COUNT(*) FROM comment_likes WHERE comment_id = ? AND is_like = 1", commentID).Scan(&out.Likes)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM comment_likes WHERE comment_id = ? AND is_like = 0", commentID).Scan(&out.Dislikes)
	out.HasReaction = hasReaction
	out.CurrentIsLike = currentIsLike

	var postID int
	_ = r.db.QueryRow("SELECT post_id FROM comments WHERE id = ?", commentID).Scan(&postID)
	return out, postID, nil
}

func (r *ForumRepo) DeleteCommentOwnedBy(userID int, commentID int) error {
	var commentAuthorID int
	if err := r.db.QueryRow("SELECT user_id FROM comments WHERE id = ?", commentID).Scan(&commentAuthorID); err != nil {
		return err
	}
	if commentAuthorID != userID {
		return sql.ErrNoRows
	}
	_, err := r.db.Exec("DELETE FROM comments WHERE id = ?", commentID)
	return err
}

