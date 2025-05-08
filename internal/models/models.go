package models

import "time"

// Pin представляет структуру данных для пина
type Pin struct {
	ID          int       `json:"id"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	UserID      int       `json:"user_id"`
	Original    *string   `json:"original"`
	Comment     *bool     `json:"comment"`
	Ai          *bool     `json:"ai"`
	Type        *string   `json:"type"`
	Title       string    `json:"title"`
	Width       *int      `json:"width"`
	Height      *int      `json:"height"`
	Duration    *float64  `json:"duration"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Comments    []Comment `gorm:"foreignKey:PinID"` // Связь с комментариями
}

// UploadPinRequest представляет данные, необходимые для загрузки нового пина
type UploadPinRequest struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Link          string `json:"link"`
	Tags          string `json:"tags"`
	AllowComments bool   `json:"allow_comments"`
	IsAiGenerated bool   `json:"is_ai_generated"`
	Media         []byte `json:"media"` // Или можно обрабатывать через multipart/form-data
	MediaHeader   struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
	} `json:"media_header"`
}

// User представляет структуру данных для пользователя
type User struct {
	ID           int        `json:"id"`
	Nickname     string     `json:"nickname"`
	Description  string     `json:"description"`
	Hidden       bool       `json:"hidden"`
	Private      bool       `json:"private"`
	Verification bool       `json:"verification"`
	Name         string     `json:"name"`
	Surname      string     `json:"surname"`
	Birth        *time.Time `json:"birth"`
	Sex          string     `json:"sex"`
	Country      *string    `json:"country"`
	Lang         *string    `json:"lang"`
	Mentions     *string    `json:"mentions"`
	Comment      bool       `json:"comment"`
	Autoplay     bool       `json:"autoplay"`
	TwoFa        bool       `json:"2fa"`
	Email        string     `json:"email"`
	Password     string     `json:"password"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Action представляет связь между пользователем и пином при лайке
type UserAction struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	PinID     int       `json:"pin_id"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

// UserSubscription представляет подписку одного пользователя на другого
type UserSubscription struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	TargetUserID int       `json:"target_user_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// Comment представляет модель комментария
type Comment struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	PinID     int       `json:"pin_id"`
	Content   string    `json:"content"`
	ReplyToID *int      `json:"reply_to_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// User      User      `gorm:"foreignKey:UserID"`
	// Replies   []Comment `gorm:"foreignKey:ReplyToID"`
}
