package queryhelper

import (
	"AREDL/demonlist"
	"AREDL/names"
	"github.com/pocketbase/pocketbase/tools/types"
)

type BaseRecord struct {
	Id      string          `db:"id" json:"id,omitempty"`
	Created *types.DateTime `db:"created" json:"created,omitempty"`
	Updated *types.DateTime `db:"updated" json:"updated,omitempty"`
}

type User struct {
	BaseRecord
	Username       string `db:"username" json:"username,omitempty"`
	Email          string `db:"email" json:"email,omitempty"`
	GlobalName     string `db:"global_name" json:"global_name,omitempty"`
	Role           string `db:"role" json:"role,omitempty"`
	Description    string `db:"description" json:"description,omitempty"`
	Country        string `db:"country" json:"country,omitempty"`
	Badges         string `db:"badges" json:"badges,omitempty"`
	AredlVerified  bool   `db:"aredl_verified" json:"aredl_verified,omitempty"`
	AredlPlus      bool   `db:"aredl_plus" json:"aredl_plus,omitempty"`
	GDUsername     string `db:"gd_username" json:"gd_username,omitempty"`
	BannedFromList bool   `db:"banned_from_list" json:"banned_from_list,omitempty"`
	Placeholder    bool   `db:"placeholder" json:"placeholder,omitempty"`
	DiscordId      string `db:"discord_id" json:"discord_id,omitempty"`
	AvatarUrl      string `db:"avatar_url" json:"avatar_url,omitempty"`
	BannerColor    string `db:"banner_color" json:"banner_color,omitempty"`
	Joined         string `db:"created" json:"joined,omitempty"`
}

func (User) TableName() string {
	return names.TableUsers
}

type AredlLevel struct {
	BaseRecord
	Position          int     `db:"position" json:"position,omitempty"`
	Name              string  `db:"name" json:"name,omitempty"`
	Publisher         *User   `db:"publisher" json:"publisher,omitempty"`
	Points            float64 `db:"points" json:"points,omitempty"`
	Legacy            bool    `db:"legacy" json:"legacy,omitempty"`
	LevelId           int     `db:"level_id" json:"level_id,omitempty"`
	LevelPassword     string  `db:"level_password" json:"level_password,omitempty"`
	CustomSong        string  `db:"custom_song" json:"custom_song,omitempty"`
	QualifyingPercent int     `db:"qualifying_percent" json:"qualifying_percent,omitempty"`
}

func (AredlLevel) TableName() string {
	return demonlist.Aredl().LevelTableName
}

type AredlSubmission struct {
	BaseRecord
	Status          string      `db:"status" json:"status,omitempty"`
	Level           *AredlLevel `db:"level" json:"level,omitempty"`
	SubmittedBy     *User       `db:"submitted_by" json:"submitted_by,omitempty"`
	VideoUrl        string      `db:"video_url" json:"video_url,omitempty"`
	Fps             int         `db:"fps" json:"fps,omitempty"`
	Mobile          bool        `db:"mobile" json:"mobile,omitempty"`
	LdmId           int         `db:"ldm_id" json:"ldm_id,omitempty"`
	PlacementOrder  int         `db:"placement_order" json:"placement_order,omitempty"`
	RawFootage      string      `db:"raw_footage" json:"raw_footage,omitempty"`
	Reviewer        *User       `db:"reviewer" json:"reviewer,omitempty"`
	RejectionReason string      `db:"rejection_reason" json:"rejection_reason,omitempty"`
}

func (AredlSubmission) TableName() string {
	return demonlist.Aredl().SubmissionTableName
}

type HistoryEntry struct {
	BaseRecord
	Level       *AredlLevel `db:"level" json:"level,omitempty"`
	Action      string      `db:"action" json:"action,omitempty"`
	NewPosition int         `db:"new_position" json:"new_position,omitempty"`
	Cause       *AredlLevel `db:"cause" json:"cause,omitempty"`
	ActionBy    *User       `db:"action_by" json:"action_by,omitempty"`
}

func (HistoryEntry) TableName() string {
	return demonlist.Aredl().HistoryTableName
}

type LeaderboardEntry struct {
	BaseRecord
	User   *User   `db:"user" json:"user,omitempty"`
	Points float64 `db:"points" json:"points,omitempty"`
	Rank   int     `db:"rank" json:"rank,omitempty"`
}

func (LeaderboardEntry) TableName() string {
	return demonlist.Aredl().LeaderboardTableName
}

type Pack struct {
	BaseRecord
	Name           string  `db:"name" json:"name,omitempty"`
	Color          string  `db:"color" json:"color,omitempty"`
	PlacementOrder int     `db:"placement_order" json:"placement_order,omitempty"`
	Points         float64 `db:"points" json:"points,omitempty"`
}

func (Pack) TableName() string {
	return demonlist.Aredl().Packs.PackTableName
}
