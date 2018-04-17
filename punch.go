package main

import (
	"time"
	"database/sql"
)

type Punch struct {
	id    int       `db="punch_id"`
	BID   int       `db="booking_id" json="BID"`
	Stamp time.Time `db="punch" json="stamp"`
}

/* Implement sorting interface (ascending sort by time) */
type Punches []Punch

func (s Punches) Less(i, j int) bool { return s[i].Stamp.Before(s[j].Stamp) }
func (s Punches) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Punches) Len() int           { return len(s) }

// Insert a punch into the DB where the punch time is given
// Returns a copy of the punch, with the id set
func (p Punch) insert() (Punch, error) {
	q := `INSERT INTO punches (booking_id, punch)
			VALUES ($1, $2) RETURNING punch_id`
	err = db.QueryRow(q, p.BID, p.Stamp).Scan(&p.id)
	return p, err
}

// Returns a slice of Punches, which corresponds to the given booking ID
func getBookingPunches(bid int) (punches Punches, err error) {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	var punches []Punch // Need to actually allocate the memory for slice by using the direct []Punch type because the compiler is silly
	q := `SELECT * FROM punches 
			WHERE booking_id = $1
			ORDER BY punch`

	stmt, err := tx.Preparex(q)
	if err != nil {
		logger.Println(err)
		return err
	}
	defer stmt.Close()
	if err = stmt.Select(&punches, bid); err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		logger.Println(err)
	}
	return punches, nil
}
