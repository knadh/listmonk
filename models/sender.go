package models

import (
    null "gopkg.in/volatiletech/null.v6"
)

// Sender represents an e-mail sender that can be verified.
type Sender struct {
    Base

    Email            string      `db:"email" json:"email"`
    Name             string      `db:"name" json:"name"`
    Verified         bool        `db:"verified" json:"verified"`
    VerificationCode null.String `db:"verification_code" json:"verification_code,omitempty"`
}
