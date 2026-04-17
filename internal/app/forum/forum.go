package forum

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/ndanbaev/forum/internal/app/ports"
)

var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not found")
	ErrBadRequest       = errors.New("bad request")
)

type Service struct {
	Repo   ports.ForumRepository
	Notify ports.NotificationService // optional
	Now    func() time.Time
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) Categories() ([]ports.Category, error) {
	return s.Repo.ListCategories()
}

type ListPostsParams struct {
	UserID            int
	Page              int
	PageSize          int
	SelectedCategories []int
	Filter            string // "", "myposts", "liked"
}

type PostFeedItem struct {
	ID            int
	UserID        int
	Title         string
	Content       string
	CreatedAtRaw  string
	Author        string
	AuthorAvatar  string
	Likes         int
	Dislikes      int
	CommentCount  int
	Categories    []ports.Category
}

type ListPostsResult struct {
	Posts      []PostFeedItem
	Page       int
	TotalPages int
	TotalPosts int
}

func (s *Service) ListPosts(p ListPostsParams) (ListPostsResult, error) {
	res, err := s.Repo.ListPosts(ports.ForumListPostsParams{
		UserID:             p.UserID,
		Page:               p.Page,
		PageSize:           p.PageSize,
		SelectedCategories: p.SelectedCategories,
		Filter:             p.Filter,
	})
	if err != nil {
		return ListPostsResult{}, err
	}
	out := ListPostsResult{Page: res.Page, TotalPages: res.TotalPages, TotalPosts: res.TotalPosts}
	for _, row := range res.Posts {
		out.Posts = append(out.Posts, PostFeedItem{
			ID:           row.ID,
			UserID:       row.UserID,
			Title:        row.Title,
			Content:      row.Content,
			CreatedAtRaw: row.CreatedAtRaw,
			Author:       row.Author,
			AuthorAvatar: row.AuthorAvatar,
			Likes:        row.Likes,
			Dislikes:     row.Dislikes,
			CommentCount: row.CommentCount,
			Categories:   row.Categories,
		})
	}
	return out, nil
}

type PostDetail struct {
	ID          int
	UserID      int
	Title       string
	Content     string
	CreatedAtRaw string
	Author      string
	AuthorAvatar string
	Likes       int
	Dislikes    int
	Categories  []ports.Category
}

type CommentDetail struct {
	ID          int
	PostID      int
	UserID      int
	Content     string
	CreatedAtRaw string
	Author      string
	AuthorAvatar string
	Likes       int
	Dislikes    int
}

type GetPostResult struct {
	Post     PostDetail
	Comments []CommentDetail
}

func (s *Service) GetPost(postID int) (GetPostResult, error) {
	res, err := s.Repo.GetPost(postID)
	if err != nil {
		if err == sql.ErrNoRows {
			return GetPostResult{}, ErrNotFound
		}
		return GetPostResult{}, err
	}
	out := GetPostResult{
		Post: PostDetail{
			ID:           res.Post.ID,
			UserID:       res.Post.UserID,
			Title:        res.Post.Title,
			Content:      res.Post.Content,
			CreatedAtRaw: res.Post.CreatedAtRaw,
			Author:       res.Post.Author,
			AuthorAvatar: res.Post.AuthorAvatar,
			Likes:        res.Post.Likes,
			Dislikes:     res.Post.Dislikes,
			Categories:   res.Post.Categories,
		},
	}
	for _, c := range res.Comments {
		out.Comments = append(out.Comments, CommentDetail{
			ID:           c.ID,
			PostID:       c.PostID,
			UserID:       c.UserID,
			Content:      c.Content,
			CreatedAtRaw: c.CreatedAtRaw,
			Author:       c.Author,
			AuthorAvatar: c.AuthorAvatar,
			Likes:        c.Likes,
			Dislikes:     c.Dislikes,
		})
	}
	return out, nil
}

func (s *Service) CreatePost(userID int, title, content string, categoryIDs []int) (int64, error) {
	if userID <= 0 {
		return 0, ErrNotAuthenticated
	}
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" || len(categoryIDs) == 0 {
		return 0, ErrBadRequest
	}
	if len([]rune(title)) > 200 || len([]rune(content)) > 10000 {
		return 0, ErrBadRequest
	}
	return s.Repo.CreatePost(userID, title, content, categoryIDs, s.now())
}

func (s *Service) AddComment(userID int, postID string, content string, actorNickname string) (commentID int64, err error) {
	if userID <= 0 {
		return 0, ErrNotAuthenticated
	}
	content = strings.TrimSpace(content)
	if postID == "" || content == "" {
		return 0, ErrBadRequest
	}
	if len(content) > 500 {
		return 0, ErrBadRequest
	}
	pi, convErr := strconv.Atoi(postID)
	if convErr != nil || pi <= 0 {
		return 0, ErrBadRequest
	}
	commentID, err = s.Repo.AddComment(userID, pi, content, s.now())
	if err != nil {
		return 0, err
	}
	postOwnerID, postTitle, _ := s.Repo.GetPostOwnerAndTitle(pi)
	if s.Notify != nil && postOwnerID > 0 && postOwnerID != userID {
		_ = s.Notify.Create(
			postOwnerID,
			userID,
			"comment",
			"New comment on your post",
			actorNickname+" commented on \""+postTitle+"\"",
			"/post/"+postID,
			s.now(),
		)
	}

	return commentID, nil
}

func (s *Service) GetComment(commentID int64) (ports.ForumCommentDetailRow, error) {
	return s.Repo.GetCommentByID(commentID)
}

type ReactionResult struct {
	Likes      int
	Dislikes   int
	IsLiked    bool
	IsDisliked bool
	HasReaction bool
	CurrentIsLike bool
}

func (s *Service) TogglePostReaction(userID int, postID string, isLike bool, actorNickname string) (ReactionResult, error) {
	if userID <= 0 {
		return ReactionResult{}, ErrNotAuthenticated
	}
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return ReactionResult{}, ErrBadRequest
	}
	pi, err := strconv.Atoi(postID)
	if err != nil || pi <= 0 {
		return ReactionResult{}, ErrBadRequest
	}
	rr, err := s.Repo.TogglePostReaction(userID, pi, isLike)
	if err != nil {
		return ReactionResult{}, err
	}
	out := ReactionResult{
		Likes:         rr.Likes,
		Dislikes:      rr.Dislikes,
		HasReaction:   rr.HasReaction,
		CurrentIsLike: rr.CurrentIsLike,
	}
	out.IsLiked = out.HasReaction && out.CurrentIsLike
	out.IsDisliked = out.HasReaction && !out.CurrentIsLike

	if out.HasReaction && s.Notify != nil {
		postOwnerID, postTitle, _ := s.Repo.GetPostOwnerAndTitle(pi)
		if postOwnerID > 0 && postOwnerID != userID {
			if out.CurrentIsLike {
				_ = s.Notify.Create(postOwnerID, userID, "like_post", "Your post got a like", actorNickname+" liked your post: "+postTitle, "/post/"+postID, s.now())
			} else {
				_ = s.Notify.Create(postOwnerID, userID, "dislike_post", "Your post got a dislike", actorNickname+" disliked your post: "+postTitle, "/post/"+postID, s.now())
			}
		}
	}

	return out, nil
}

func (s *Service) ToggleCommentReaction(userID int, commentID string, isLike bool, actorNickname string) (ReactionResult, error) {
	if userID <= 0 {
		return ReactionResult{}, ErrNotAuthenticated
	}
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		return ReactionResult{}, ErrBadRequest
	}
	ci, err := strconv.Atoi(commentID)
	if err != nil || ci <= 0 {
		return ReactionResult{}, ErrBadRequest
	}
	var rr ports.ForumReactionResult
	var postIDForLink int
	rr, postIDForLink, err = s.Repo.ToggleCommentReaction(userID, ci, isLike)
	if err != nil {
		return ReactionResult{}, err
	}
	out := ReactionResult{
		Likes:         rr.Likes,
		Dislikes:      rr.Dislikes,
		HasReaction:   rr.HasReaction,
		CurrentIsLike: rr.CurrentIsLike,
	}
	out.IsLiked = out.HasReaction && out.CurrentIsLike
	out.IsDisliked = out.HasReaction && !out.CurrentIsLike

	if out.HasReaction && s.Notify != nil {
		row, err2 := s.Repo.GetCommentByID(int64(ci))
		if err2 == nil {
			commentOwnerID := row.UserID
			if commentOwnerID > 0 && commentOwnerID != userID {
				if out.CurrentIsLike {
					_ = s.Notify.Create(commentOwnerID, userID, "like_comment", "Your comment got a like", actorNickname+" liked your comment", "/post/"+strconv.Itoa(postIDForLink), s.now())
				} else {
					_ = s.Notify.Create(commentOwnerID, userID, "dislike_comment", "Your comment got a dislike", actorNickname+" disliked your comment", "/post/"+strconv.Itoa(postIDForLink), s.now())
				}
			}
		}
	}

	return out, nil
}

func (s *Service) DeleteComment(userID int, commentID string) error {
	if userID <= 0 {
		return ErrNotAuthenticated
	}
	if strings.TrimSpace(commentID) == "" {
		return ErrBadRequest
	}
	ci, err := strconv.Atoi(strings.TrimSpace(commentID))
	if err != nil || ci <= 0 {
		return ErrBadRequest
	}
	err = s.Repo.DeleteCommentOwnedBy(userID, ci)
	if err != nil {
		// Not found vs forbidden is inferred from the DB error surface.
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return ErrForbidden
	}
	return nil
}

