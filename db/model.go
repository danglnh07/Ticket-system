package db

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type AccountStatus string

type Role string

type OauthProvider string

type EventStatus string

type TicketStatus string

const (
	Inactive AccountStatus = "inactive"
	Active   AccountStatus = "active"
	Banned   AccountStatus = "banned"

	Admin          Role = "admin"
	Organiser      Role = "organiser"
	SupportedStaff Role = "staff"
	User           Role = "user"

	Google   OauthProvider = "google"
	Telegram OauthProvider = "telegram"

	Draft     EventStatus = "draft"
	Published EventStatus = "published"
	Canceled  EventStatus = "canceled"

	Pending TicketStatus = "pending"
	Valid   TicketStatus = "valid"
	Used    TicketStatus = "used"
	Expired TicketStatus = "expired"
	Refund  TicketStatus = "refund"
)

type Account struct {
	gorm.Model

	// Username and email should be indexes since they rarely changed, and we query them frequently
	Username string `json:"username" gorm:"not null;unique;uniqueIndex"`
	Email    string `json:"email" gorm:"not null; unique;uniqueIndex"`

	// If user login via OAuth2, then password would be null
	Password sql.NullString `json:"password"`

	// The link to access avatar
	Avatar string `json:"avatar"`

	// Status and role values rarely change, so we fix them in code instead of create new tables
	Status AccountStatus `json:"status" gorm:"not null"`
	Role   Role          `json:"role" gorm:"not null"`

	// Point system, initial value as 0
	Point uint `json:"point"`

	// OAuth2 credential, include access token and refresh token if integrate with OAuth service
	OauthProvider     sql.NullString `json:"oauth_provider"`
	OauthProviderID   sql.NullString `json:"oauth_provider_id"`
	OauthAccessToken  sql.NullString `json:"oauth_access_token"`
	OauthRefreshToken sql.NullString `json:"oauth_refresh_token"`

	// This is the token version of internal tokens
	TokenVersion uint `json:"token_version" gorm:"not null"`
}

type Membership struct {
	gorm.Model

	// Tier of the membership: bronze, silver, gold,...
	Tier string `json:"tier" gorm:"not null"`

	// The minimum point to be at this tier
	BasePoint uint `json:"base_point" gorm:"not null"`

	// Discount for each tier (in %)
	Discount uint `json:"discount"`
}

type Event struct {
	gorm.Model

	// The host (creator) of the event
	HostID uint    `json:"host_id" gorm:"not null"`
	Host   Account `json:"host" gorm:"foreignKey:HostID"`

	// Event information
	Name         string         `json:"name" gorm:"not null"`
	Description  string         `json:"description" gorm:"not null"`
	Location     string         `json:"location" gorm:"not null"`
	StartTime    time.Time      `json:"start_time" gorm:"not null"`
	EndTime      time.Time      `json:"end_time" gorm:"not null"`
	PreviewImage sql.NullString `json:"preview_image"`
	Status       EventStatus    `json:"status" gorm:"not null"`

	// Google Calendar Event ID
	CalendarID sql.NullString `json:"calendar_id"`
}

type Ticket struct {
	gorm.Model

	// The event that issue this ticket
	EventID uint  `json:"event_id" gorm:"not null"`
	Event   Event `json:"event" gorm:"foreignKey:EventID"`

	// The rank of the ticket: standard, VIP,...
	Rank string `json:"rank" gorm:"not null"`

	// The total ticket that host issues
	Total uint `json:"total" gorm:"total"`

	// The remaining tickets
	Available uint `json:"available" gorm:"not null"`

	// The price of the ticket (without applying memebership discount)
	Price float64 `json:"price" gorm:"not null"`

	// This is the global status of all tickets of a type belong to a specific event
	// So its status would be similar to event status (draft, published, canceled).
	// Local status (status of each ticket after user has bought it) is different. It would be: used, canceled. refund,...
	Status EventStatus `json:"status" gorm:"not null"`
}

type Booking struct {
	gorm.Model

	// Buyer of the ticket
	AccountID uint    `json:"account_id" gorm:"not null"`
	Account   Account `json:"account" gorm:"foreignKey:AccountID"`

	// The ticket type
	TicketID uint   `json:"ticket_id" gorm:"not null"`
	Ticket   Ticket `json:"ticket" gorm:"foreignKey:TicketID"`

	// Seat number. Since the event location is not managed by the system,
	// there is no constraint that can be applied to seat number
	SeatNumber string `json:"seat_number" gorm:"not null"`

	// Ticket status: pending (has booked, but not pay), valid (has payed, has not used),
	// used, expired (valid, not used even after event ended), refund (event canceled -> ticket is refund)
	Status TicketStatus `json:"status" gorm:"not null"`
}
