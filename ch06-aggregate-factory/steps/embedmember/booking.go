// Package embedmember は、Booking に Member を持たせて
// 一緒に保存する例である。DB の代わりに map を使う。
package embedmember

// Member は会員である。
type Member struct {
	ID    string
	Email string
}

// #region embed

// Booking は会員そのものを持つ予約である。
type Booking struct {
	ID     string
	Member *Member
	Status string
}

// #endregion

// Store は読み書きのたびに値をコピーする、DB の代わりである。
type Store struct {
	members  map[string]Member
	bookings map[string]Booking
}

// NewStore は会員と予約を1件ずつ持つ Store を返す。
func NewStore(m Member, bookingID string) *Store {
	return &Store{
		members: map[string]Member{m.ID: m},
		bookings: map[string]Booking{
			bookingID: {ID: bookingID, Status: "pending"},
		},
	}
}

// Load は予約を、会員と一緒に読み込む。
func (s *Store) Load(id string, memberID string) Booking {
	b := s.bookings[id]
	m := s.members[memberID]
	b.Member = &m
	return b
}

// Member は会員を読み込む。
func (s *Store) Member(id string) Member { return s.members[id] }

// SaveMember は会員だけを保存する。
func (s *Store) SaveMember(m Member) { s.members[m.ID] = m }

// #region save

// Save は予約と、予約が持つ会員を保存する。
func (s *Store) Save(b Booking) {
	s.bookings[b.ID] = b
	s.members[b.Member.ID] = *b.Member
}

// #endregion

// #region by-id

// BookingByID は会員を ID で指す予約である。
type BookingByID struct {
	ID       string
	MemberID string
	Status   string
}

// SaveByID は予約だけを保存する。会員には触れない。
func (s *Store) SaveByID(b BookingByID) {
	s.bookings[b.ID] = Booking{ID: b.ID, Status: b.Status}
}

// #endregion
