package ports

import "time"

// RealtimePublisher publishes realtime events to connected clients.
// Implementations should preserve WS message types/payloads expected by the frontend.
type RealtimePublisher interface {
	BroadcastToUser(userID int, msgType string, payload any)
	OnlineUserIDs() []int
}

type User struct {
	ID        int
	Nickname  string
	Age       int
	Gender    string
	FirstName string
	LastName  string
	Email     string
	Password  string
	AvatarURL string
}

type Category struct {
	ID   int
	Name string
}

type Session struct {
	UserID  int
	UUID    string
	Expires time.Time
}

type UserRepository interface {
	CountByEmailLower(email string) (int, error)
	CountByNicknameLower(nickname string) (int, error)

	GetByID(id int) (*User, error)
	GetAuthByEmailLower(email string) (id int, passwordHash string, err error)
	GetAuthByNicknameLower(nickname string) (id int, passwordHash string, err error)
	GetAuthByUsernameLower(username string) (id int, passwordHash string, err error)
	GetNicknameByID(id int) (string, error)
	CreateUser(nickname string, age int, gender, firstName, lastName, email, passwordHash string) error
	CreateUserLegacyUsername(nickname string, age int, gender, firstName, lastName, email, passwordHash string) error
	UpdatePasswordHashByID(id int, passwordHash string) error

	GetAvatarURLByID(id int) (string, error)
	UpdateAvatarURLByID(id int, avatarURL string) error
}

type AvatarStorage interface {
	SaveAvatar(userID int, data []byte, contentType string) (relativeURL string, fullPath string, err error)
	DeleteFile(path string) error
}

type SessionRepository interface {
	GetUserIDByUUID(uuid string) (userID int, expires time.Time, err error)
	DeleteByUserID(userID int) error
	DeleteByUUID(uuid string) error
	Insert(userID int, uuid string, expires time.Time) error
}

type NotificationRepository interface {
	Insert(userID, actorID int, notifType, title, message, link string, createdAt time.Time) error
	ListByUser(userID int, limit int) ([]NotificationItem, error)
	CountUnread(userID int) (int, error)
	MarkRead(userID int, notificationID int) error
	MarkAllRead(userID int) error
}

type NotificationItem struct {
	ID        int
	Type      string
	Title     string
	Message   string
	Link      string
	IsRead    bool
	CreatedAt time.Time
}

type NotificationService interface {
	Create(userID, actorID int, notifType, title, message, link string, createdAt time.Time) error
}

type ForumRepository interface {
	ListCategories() ([]Category, error)

	ListPosts(p ForumListPostsParams) (ForumListPostsResult, error)
	GetPost(postID int) (ForumGetPostResult, error)
	CreatePost(userID int, title, content string, categoryIDs []int, createdAt time.Time) (int64, error)

	AddComment(userID int, postID int, content string, createdAt time.Time) (int64, error)
	GetCommentByID(commentID int64) (ForumCommentDetailRow, error)
	GetPostOwnerAndTitle(postID int) (ownerID int, title string, err error)

	TogglePostReaction(userID int, postID int, isLike bool) (res ForumReactionResult, err error)
	ToggleCommentReaction(userID int, commentID int, isLike bool) (res ForumReactionResult, postIDForLink int, err error)

	DeleteCommentOwnedBy(userID int, commentID int) error
}

type ForumListPostsParams struct {
	UserID            int
	Page              int
	PageSize          int
	SelectedCategories []int
	Filter            string
}

type ForumPostFeedItemRow struct {
	ID           int
	UserID       int
	Title        string
	Content      string
	CreatedAtRaw string
	Author       string
	AuthorAvatar string
	Likes        int
	Dislikes     int
	CommentCount int
	Categories   []Category
}

type ForumListPostsResult struct {
	Posts      []ForumPostFeedItemRow
	Page       int
	TotalPages int
	TotalPosts int
}

type ForumPostDetailRow struct {
	ID            int
	UserID        int
	Title         string
	Content       string
	CreatedAtRaw  string
	Author        string
	AuthorAvatar  string
	Likes         int
	Dislikes      int
	Categories    []Category
}

type ForumCommentDetailRow struct {
	ID           int
	PostID       int
	UserID       int
	Content      string
	CreatedAtRaw string
	Author       string
	AuthorAvatar string
	Likes        int
	Dislikes     int
}

type ForumGetPostResult struct {
	Post     ForumPostDetailRow
	Comments []ForumCommentDetailRow
}

type ForumReactionResult struct {
	Likes        int
	Dislikes     int
	HasReaction  bool
	CurrentIsLike bool
}

type MessageRepository interface {
	InsertPrivateMessage(fromID, toID int, content string, createdAt time.Time) error
	ListConversationUsersWithPreview(userID int) ([]ConversationRow, error)
	ListMessagesBetween(userID, otherID int, limit int, beforeID int) ([]PrivateMessageRow, bool, error)
}

type ConversationRow struct {
	UserID         int
	Nickname       string
	AvatarURL      string
	LastAt         *time.Time
	LastMsgPreview *string
}

type PrivateMessageRow struct {
	ID        int
	FromID    int
	ToID      int
	Content   string
	CreatedAt time.Time
	FromName  string
}

