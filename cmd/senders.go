package main

import (
    "crypto/rand"
    "database/sql"
    "encoding/hex"
    "fmt"
    "net/http"
    "strings"

    "github.com/knadh/listmonk/models"
    "github.com/labstack/echo/v4"
)

// createSenderReq is the payload for creating a new sender.
type createSenderReq struct {
    Email string `json:"email"`
    Name  string `json:"name"`
}

// CreateSender creates a new sender row with a verification code and sends
// a verification e-mail to the given address.
func (a *App) CreateSender(c echo.Context) error {
    var req createSenderReq
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
    }
    req.Email = strings.TrimSpace(req.Email)
    if req.Email == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "email required"})
    }

    // generate a secure random code
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "could not generate code"})
    }
    code := hex.EncodeToString(b)

    // upsert sender with new code and set verified=false
    query := `INSERT INTO senders (email, name, verification_code, verified, created_at, updated_at)
              VALUES ($1, $2, $3, false, NOW(), NOW())
              ON CONFLICT (email) DO UPDATE SET verification_code = EXCLUDED.verification_code, verified = false, updated_at = NOW()`
    if _, err := a.db.Exec(query, req.Email, req.Name, code); err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "db error"})
    }

    // build a simple verification link and plain-text body
    link := fmt.Sprintf("%s/api/senders/verify?email=%s&code=%s", strings.TrimRight(a.urlCfg.RootURL, "/"), req.Email, code)
    subject := "Verify sender address"
    body := fmt.Sprintf("Please verify this sender address by clicking the link below:\n\n%s\n\nOr use this code: %s\n", link, code)

    // send using the configured email messenger (app.emailMsgr)
    m := models.Message{
        Messenger:   "email",
        ContentType: models.CampaignContentTypePlain,
        From:        a.cfg.FromEmail,
        To:          []string{req.Email},
        Subject:     subject,
        Body:        []byte(body),
    }

    if a.emailMsgr == nil {
        // not initialized; return success for creation but warn in response
        return c.JSON(http.StatusOK, map[string]string{"status": "created", "note": "verification e-mail not sent (email messenger not configured)"})
    }

    if err := a.emailMsgr.Push(m); err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed sending verification e-mail"})
    }

    return c.JSON(http.StatusOK, map[string]string{"status": "created"})
}

// verifySenderReq is the payload to verify a sender.
type verifySenderReq struct {
    Email string `json:"email"`
    Code  string `json:"code"`
}

// VerifySender verifies a sender with the given code.
func (a *App) VerifySender(c echo.Context) error {
    // Accept both query params (GET) and JSON body (POST)
    email := strings.TrimSpace(c.QueryParam("email"))
    code := strings.TrimSpace(c.QueryParam("code"))
    if email == "" || code == "" {
        var req verifySenderReq
        if err := c.Bind(&req); err != nil {
            return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and code required"})
        }
        email = strings.TrimSpace(req.Email)
        code = strings.TrimSpace(req.Code)
    }
    if email == "" || code == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and code required"})
    }

    var dbCode sql.NullString
    var id int
    err := a.db.QueryRow("SELECT id, verification_code FROM senders WHERE LOWER(email)=LOWER($1)", email).Scan(&id, &dbCode)
    if err != nil {
        if err == sql.ErrNoRows {
            return c.JSON(http.StatusNotFound, map[string]string{"error": "sender not found"})
        }
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "db error"})
    }

    if !dbCode.Valid || dbCode.String != code {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid code"})
    }

    if _, err := a.db.Exec("UPDATE senders SET verified = true, verification_code = NULL, updated_at = NOW() WHERE id = $1", id); err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "db update error"})
    }

    return c.JSON(http.StatusOK, map[string]string{"status": "verified"})
}
